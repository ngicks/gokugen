package taskstorage

import (
	"errors"
	"fmt"
	"time"

	"github.com/ngicks/gokugen"
)

var (
	ErrMiddlewareOrder   = errors.New("invalid middleware order")
	ErrNonexistentWorkId = errors.New("nonexistent work id")
)

type WorkFn = gokugen.WorkFn
type WorkFnWParam = gokugen.WorkFnWParam

type WorkRegistry interface {
	Load(key string) (value WorkFnWParam, ok bool)
}

type SingleNodeTaskStorage struct {
	repo          Repository
	failedIds     *SyncStateStore
	shouldRestore func(TaskInfo) bool
	workRegistry  WorkRegistry
	taskMap       *TaskMap
}

func NewSingleNodeTaskStorage(
	repo Repository,
	shouldRestore func(TaskInfo) bool,
	workRegistry WorkRegistry,
) *SingleNodeTaskStorage {
	return &SingleNodeTaskStorage{
		repo:          repo,
		shouldRestore: shouldRestore,
		failedIds:     NewSyncStateStore(),
		workRegistry:  workRegistry,
		taskMap:       NewTaskMap(),
	}
}

func (ts *SingleNodeTaskStorage) paramLoad(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		taskStorage, ok := ctx.(*taskStorageCtx)
		if !ok {
			return nil, fmt.Errorf("%w: paramLoad must be right after storeTask.", ErrMiddlewareOrder)
		}

		taskId, _ := GetTaskId(taskStorage)
		workWithParam, ok := ts.workRegistry.Load(taskStorage.workId)
		if !ok {
			return nil, fmt.Errorf("%w: unknown work id = %s", ErrNonexistentWorkId, taskStorage.workId)

		}
		paramLoadable := newtaskStorageParamLoadableCtx(
			taskStorage,
			func() (any, error) {
				info, err := ts.repo.GetById(taskId)
				if err != nil {
					return nil, err
				}
				return info.Param, nil
			},
			workWithParam,
			func(err error) {
				markDoneTask(err, ts, taskId)
			},
		)
		return handler(paramLoadable)
	}
}

func (ts *SingleNodeTaskStorage) storeTask(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (task gokugen.Task, err error) {
		param, err := GetParam(ctx)
		if err != nil {
			return
		}
		workId, err := GetWorkId(ctx)
		if err != nil {
			return
		}
		scheduledTime := ctx.ScheduledTime()
		workRaw, ok := ts.workRegistry.Load(workId)
		if !ok {
			err = ErrNonexistentWorkId
			return
		}

		taskId, err := ts.repo.Insert(TaskInfo{
			WorkId:        workId,
			Param:         param,
			ScheduledTime: scheduledTime,
			State:         Initialized,
		})
		if err != nil {
			return
		}

		ok = setWork(ctx, func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
			err := workRaw(ctxCancelCh, taskCancelCh, scheduled, param)
			markDoneTask(err, ts, taskId)
			if err != nil {
				return err
			}
			return nil
		})
		if !ok {
			panic("SingleNodeTaskStorage: implementation error. setWork returned false")
		}

		ok = setTaskId(ctx, taskId)
		if !ok {
			panic("SingleNodeTaskStorage: implementation error. setTaskId returned false")
		}

		task, err = handler(ctx)
		if err != nil {
			return
		}
		task = wrapCancel(ts.repo, ts.failedIds, taskId, task)
		ts.taskMap.LoadOrStore(taskId, task)
		return
	}
}

func markDoneTask(result error, ts *SingleNodeTaskStorage, taskId string) {
	ts.taskMap.Delete(taskId)
	if errors.Is(result, ErrOtherNodeWorkingOnTheTask) {
		return
	} else if result != nil {
		_, err := ts.repo.MarkAsFailed(taskId)
		if err != nil {
			ts.failedIds.Put(taskId, Failed)
		}
	} else {
		_, err := ts.repo.MarkAsDone(taskId)
		if err != nil {
			ts.failedIds.Put(taskId, Done)
		}
	}
}

func (ts *SingleNodeTaskStorage) Middleware(freeParam bool) []gokugen.MiddlewareFunc {
	if freeParam {
		return []gokugen.MiddlewareFunc{ts.storeTask, ts.paramLoad}
	}
	return []gokugen.MiddlewareFunc{ts.storeTask}
}

func (s *SingleNodeTaskStorage) Sync(schedule func(ctx gokugen.SchedulerContext) (gokugen.Task, error)) (restored bool, rescheduled map[string]gokugen.Task, err error) {
	fetchedIds, err := s.repo.GetAll()
	if err != nil {
		return
	}
	rescheduled = make(map[string]gokugen.Task)
	knownIds := s.taskMap.Clone()
	for _, fetched := range fetchedIds {
		had := knownIds.Delete(fetched.Id)
		if had || !s.shouldRestore(fetched) {
			continue
		}

		_, ok := s.workRegistry.Load(fetched.WorkId)
		if !ok {
			// log? return err?
			continue
		}

		ctx := newTaskStorageCtx(
			gokugen.NewPlainContext(fetched.ScheduledTime, nil, make(map[any]any)),
			fetched.WorkId,
			fetched.Param,
		)
		var task gokugen.Task
		task, err = schedule(ctx)
		if err != nil {
			return
		}
		restored = true
		rescheduled[fetched.Id] = task
		s.taskMap.LoadOrStore(fetched.Id, task)
	}
	removedIds := knownIds.AllIds()
	for _, id := range removedIds {
		s.taskMap.Delete(id)
	}
	return
}

func (s *SingleNodeTaskStorage) RetryMarking() (allRemoved bool) {
	for _, set := range s.failedIds.GetAll() {
		if !s.failedIds.Has(set.Key) {
			continue
		}
		var noop bool
		var err error
		switch set.Value {
		case Done:
			noop, err = s.repo.MarkAsDone(set.Key)
		case Cancelled:
			noop, err = s.repo.MarkAsCancelled(set.Key)
		case Failed:
			noop, err = s.repo.MarkAsFailed(set.Key)
		}

		if noop || err == nil {
			s.failedIds.Remove(set.Key)
		}
	}
	return s.failedIds.Len() == 0
}

type repoCancellableTaskWrapper struct {
	gokugen.Task
	id        string
	repo      Repository
	failedIds *SyncStateStore
}

func (tw *repoCancellableTaskWrapper) Cancel() (cancelled bool) {
	cancelled = tw.Task.Cancel()
	if cancelled && !tw.IsDone() {
		_, err := tw.repo.MarkAsCancelled(tw.id)
		if err != nil {
			tw.failedIds.Put(tw.id, Cancelled)
		}
	}
	return
}

func wrapCancel(repo Repository, failedIds *SyncStateStore, id string, t gokugen.Task) *repoCancellableTaskWrapper {
	return &repoCancellableTaskWrapper{
		id:   id,
		repo: repo,
		Task: t,
	}
}
