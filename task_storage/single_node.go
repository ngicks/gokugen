package taskstorage

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/common"
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
	mu            sync.Mutex
	getNow        common.GetNow // this field can be swapped out in test codes.
	lastSynced    time.Time
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
		getNow:        common.GetNowImpl{},
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

		loadable := &paramLoadableCtx{
			SchedulerContext: &baseCtx{
				SchedulerContext: ctx,
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
			wrapper: func(self gokugen.SchedulerContext, workFn WorkFn) WorkFn {
				return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
					param, err := GetParam(self)
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

		taskId, err := GetTaskId(ctx)
		if err != nil {
			return
		}

		if taskId == "" {
			taskId, err = ts.repo.Insert(TaskInfo{
				WorkId:        workId,
				Param:         param,
				ScheduledTime: scheduledTime,
				State:         Initialized,
			})
			if err != nil {
				return
			}
		}

		fnWrapped := &fnWrapperCtx{
			SchedulerContext: &baseCtx{
				SchedulerContext: ctx,
				taskId:           taskId,
				workId:           workId,
			},
			wrapper: func(self gokugen.SchedulerContext, _ WorkFn) WorkFn {
				return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
					param, err := GetParam(self)
					if err != nil {
						return err
					}
					err = workRaw(ctxCancelCh, taskCancelCh, scheduled, param)
					markDoneTask(err, ts, taskId)
					return err
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
	s.mu.Lock()
	defer s.mu.Unlock()

	fetchedIds, err := s.repo.GetUpdatedSince(s.lastSynced)
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

	s.lastSynced = s.getNow.GetNow()
	return
}

func (s *SingleNodeTaskStorage) RetryMarking() (allRemoved bool) {
	for _, set := range s.failedIds.GetAll() {
		if !s.failedIds.Remove(set.Key) {
			continue
		}
		var err error
		switch set.Value {
		case Done:
			_, err = s.repo.MarkAsDone(set.Key)
		case Cancelled:
			_, err = s.repo.MarkAsCancelled(set.Key)
		case Failed:
			_, err = s.repo.MarkAsFailed(set.Key)
		}

		if err != nil {
			s.failedIds.Put(set.Key, set.Value)
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
