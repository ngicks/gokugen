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

type ExternalStateChangeErr struct {
	id    string
	state TaskState
}

func (e ExternalStateChangeErr) Error() string {
	return fmt.Sprintf("The state is changed externally: id = %s, state = %s", e.id, e.state)
}

type WorkFn = gokugen.WorkFn
type WorkFnWParam = gokugen.WorkFnWParam

type WorkRegistry interface {
	Load(key string) (value WorkFnWParam, ok bool)
}

type SingleNodeTaskStorage struct {
	repo           Repository
	failedIds      *SyncStateStore
	shouldRestore  func(TaskInfo) bool
	workRegistry   WorkRegistry
	taskMap        *TaskMap
	mu             sync.Mutex
	getNow         common.GetNow // this field can be swapped out in test codes.
	lastSynced     time.Time
	syncCtxWrapper func(gokugen.SchedulerContext) gokugen.SchedulerContext
}

func NewSingleNodeTaskStorage(
	repo Repository,
	shouldRestore func(TaskInfo) bool,
	workRegistry WorkRegistry,
	syncCtxWrapper func(gokugen.SchedulerContext) gokugen.SchedulerContext,
) *SingleNodeTaskStorage {
	return &SingleNodeTaskStorage{
		repo:           repo,
		shouldRestore:  shouldRestore,
		failedIds:      NewSyncStateStore(),
		workRegistry:   workRegistry,
		taskMap:        NewTaskMap(),
		getNow:         common.GetNowImpl{},
		syncCtxWrapper: syncCtxWrapper,
	}
}

func (ts *SingleNodeTaskStorage) paramLoad(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		taskId, err := gokugen.GetTaskId(ctx)
		if err != nil {
			return nil, err
		}
		loadable := gokugen.WithParamLoader(
			ctx,
			func() (any, error) {
				info, err := ts.repo.GetById(taskId)
				if err != nil {
					return nil, err
				}
				return info.Param, nil
			},
		)

		return handler(loadable)
	}
}

func (ts *SingleNodeTaskStorage) storeTask(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (task gokugen.Task, err error) {
		param, err := gokugen.GetParam(ctx)
		if err != nil {
			return
		}
		workId, err := gokugen.GetWorkId(ctx)
		if err != nil {
			return
		}
		scheduledTime := ctx.ScheduledTime()
		workWithParam, ok := ts.workRegistry.Load(workId)
		if !ok {
			err = fmt.Errorf("%w: unknown work id = %s", ErrNonexistentWorkId, workId)
			return
		}

		taskId, err := gokugen.GetTaskId(ctx)
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

		fnWrapped := gokugen.WithWorkFnWrapper(
			gokugen.WithTaskId(
				ctx,
				taskId,
			),
			func(self gokugen.SchedulerContext, _ WorkFn) WorkFn {
				return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
					param, err := gokugen.GetParam(self)
					if err != nil {
						return err
					}
					err = workWithParam(ctxCancelCh, taskCancelCh, scheduled, param)
					markDoneTask(err, ts, taskId)
					return err
				}
			},
		)

		task, err = handler(fnWrapped)
		if err != nil {
			return
		}
		task = wrapCancel(ts.repo, ts.failedIds, ts.taskMap, taskId, task)
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

func (ts *SingleNodeTaskStorage) Sync(
	schedule func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
) (rescheduled map[string]gokugen.Task, schedulingErr map[string]error, err error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	syncedAt := ts.getNow.GetNow()
	fetchedIds, err := ts.repo.GetUpdatedSince(ts.lastSynced)
	if err != nil {
		return
	}

	rescheduled = make(map[string]gokugen.Task)
	schedulingErr = make(map[string]error)

	for _, fetched := range fetchedIds {
		if fetched.LastModified.After(syncedAt) {
			// Most recent time of fetched tasks is next last-synced time.
			// We do want set it correctly to avoid doubly syncing that.
			syncedAt = fetched.LastModified
		}

		_, ok := ts.workRegistry.Load(fetched.WorkId)
		if !ok {
			schedulingErr[fetched.Id] = fmt.Errorf("%w: unknown work id = %s", ErrNonexistentWorkId, fetched.WorkId)
			continue
		}

		switch fetched.State {
		case Working, Done, Cancelled, Failed:
			if inTaskMap, loaded := ts.taskMap.LoadAndDelete(fetched.Id); loaded {
				inTaskMap.CancelWithReason(ExternalStateChangeErr{
					id:    fetched.Id,
					state: fetched.State,
				})
			}
			continue
		default:
			if ts.taskMap.Has(fetched.Id) {
				continue
			}
		}

		if !ts.shouldRestore(fetched) {
			continue
		}

		param := fetched.Param
		var ctx gokugen.SchedulerContext = gokugen.WithParam(
			gokugen.WithWorkId(
				gokugen.WithTaskId(
					gokugen.NewPlainContext(fetched.ScheduledTime, nil, make(map[any]any)),
					fetched.Id,
				),
				fetched.WorkId,
			),
			param,
		)

		if ts.syncCtxWrapper != nil {
			ctx = ts.syncCtxWrapper(ctx)
		}

		var task gokugen.Task
		task, err = schedule(ctx)
		if err != nil {
			schedulingErr[fetched.Id] = err
			continue
		}
		rescheduled[fetched.Id] = task
		ts.taskMap.LoadOrStore(fetched.Id, task)
	}

	ts.lastSynced = syncedAt
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

func wrapCancel(repo Repository, failedIds *SyncStateStore, taskMap *TaskMap, id string, t gokugen.Task) gokugen.Task {
	return &taskWrapper{
		Task: t,
		cancel: func(baseCanceller func(err error) (cancelled bool)) func(err error) (cancelled bool) {
			return func(err error) (cancelled bool) {
				cancelled = baseCanceller(err)
				taskMap.Delete(id)
				if _, ok := err.(ExternalStateChangeErr); ok {
					// no marking is needed since it is already changed by external source!
					return
				}
				if cancelled && !t.IsDone() {
					// if it's done, we dont want to call heavy marking method.
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
