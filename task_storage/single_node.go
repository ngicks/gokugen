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
		taskId, err := GetTaskId(ctx)
		if err != nil {
			return nil, err
		}
		workId, err := GetWorkId(ctx)
		if err != nil {
			return nil, err
		}
		workWithParam, ok := ts.workRegistry.Load(workId)
		if !ok {
			return nil, fmt.Errorf("%w: unknown work id = %s", ErrNonexistentWorkId, workId)
		}

		var parentCtx gokugen.SchedulerContext
		parentCtx = ctx
		for {
			if !isTaskStorageCtx(parentCtx) {
				break
			}
			if unwrappable, ok := parentCtx.(interface {
				Unwrap() gokugen.SchedulerContext
			}); ok {
				parentCtx = unwrappable.Unwrap()
			}
		}

		loadable := &paramLoadableCtx{
			SchedulerContext: &baseCtx{
				SchedulerContext: parentCtx,
				taskId:           taskId,
				workId:           workId,
			},
			paramLoader: func() (any, error) {
				info, err := ts.repo.GetById(taskId)
				if err != nil {
					return nil, err
				}
				return info.Param, nil
			},
		}

		fnWrapped := &fnWrapperCtx{
			SchedulerContext: loadable,
			wrapper: func(workFn WorkFn) WorkFn {
				return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
					param, err := GetParam(loadable)
					if err != nil {
						return err
					}
					err = workWithParam(ctxCancelCh, taskCancelCh, scheduled, param)
					markDoneTask(err, ts, taskId)
					return err
				}
			},
		}

		return handler(fnWrapped)
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

		fnWrapped := &fnWrapperCtx{
			SchedulerContext: &baseCtx{
				SchedulerContext: ctx,
				taskId:           taskId,
				workId:           workId,
			},
			wrapper: func(WorkFn) WorkFn {
				return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
					err := workRaw(ctxCancelCh, taskCancelCh, scheduled, param)
					markDoneTask(err, ts, taskId)
					if err != nil {
						return err
					}
					return nil
				}
			},
		}

		task, err = handler(fnWrapped)
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

func (s *SingleNodeTaskStorage) Sync(
	schedule func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
) (restored bool, rescheduled map[string]gokugen.Task, err error) {
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

		schedTime := fetched.ScheduledTime
		workId := fetched.WorkId
		param := fetched.Param
		ctx := &paramLoadableCtx{
			SchedulerContext: &baseCtx{
				SchedulerContext: gokugen.NewPlainContext(schedTime, nil, make(map[any]any)),
				workId:           workId,
			},
			paramLoader: func() (any, error) {
				return param, nil
			},
		}
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

func wrapCancel(repo Repository, failedIds *SyncStateStore, id string, t gokugen.Task) gokugen.Task {
	return &taskWrapper{
		base: t,
		cancel: func(baseCanceller func() (cancelled bool)) func() (cancelled bool) {
			return func() (cancelled bool) {
				cancelled = baseCanceller()
				if cancelled && !t.IsDone() {
					_, err := repo.MarkAsCancelled(id)
					if err != nil {
						failedIds.Put(id, Cancelled)
					}
				}
				return
			}
		},
	}
}
