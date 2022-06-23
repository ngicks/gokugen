package taskstorage

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/common"
	"github.com/ngicks/type-param-common/set"
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

// WorkRegistry is used to retrieve work function by workId.
type WorkRegistry interface {
	Load(key string) (value WorkFnWParam, ok bool)
}

// SingleNodeTaskStorage provides ability to store task information to, and restore them from persistent data storage.
type SingleNodeTaskStorage struct {
	repo           Repository
	failedIds      *SyncStateStore
	shouldRestore  func(TaskInfo) bool
	workRegistry   WorkRegistry
	taskMap        *TaskMap
	mu             sync.Mutex
	getNow         common.GetNow // this field can be swapped out in test codes.
	lastSynced     time.Time
	knownIdForTime set.Set[string]
	syncCtxWrapper func(gokugen.SchedulerContext) gokugen.SchedulerContext
}

// NewSingleNodeTaskStorage creates new SingleNodeTaskStorage instance.
//
// repo is Repository, interface to manipulate persistent data storage.
//
// shouldRestore is used in Sync, to decide if task should be restored and re-scheduled in internal scheduler.
// (e.g. ignore tasks if they are too old and overdue.)
//
// workRegistry is used to retrieve work function associated to workId.
// User must register functions to registry beforehand.
//
// syncCtxWrapper is used in Sync. Sync tries to schedule newly craeted context.
// this context will be wrapped with syncCtxWrapper if non nil.
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
		knownIdForTime: set.Set[string]{},
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
		if err != nil && !errors.Is(err, gokugen.ErrValueNotFound) {
			return
		}

		hadTaskId := true
		if taskId == "" {
			// ctx does not contain task id.
			// needs to create new entry in repository.
			hadTaskId = false
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

		var newCtx gokugen.SchedulerContext = ctx
		if !hadTaskId {
			newCtx = gokugen.WithTaskId(
				ctx,
				taskId,
			)
		}
		if workSet := ctx.Work(); workSet == nil {
			newCtx = gokugen.WithWorkFnWrapper(
				newCtx,
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
		}

		task, err = handler(newCtx)
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

// Middleware returns slice of gokugen.MiddlewareFunc. Order must maintained.
// Though these middleware, task context info is stored in external persistent data storage.
//
// If freeParam is true, param free up functionality is enabled.
// It let those middlewares to forget param until needed. This adds one middleware
// that load up param from repository right before execution.
func (ts *SingleNodeTaskStorage) Middleware(freeParam bool) []gokugen.MiddlewareFunc {
	if freeParam {
		return []gokugen.MiddlewareFunc{ts.storeTask, ts.paramLoad}
	}
	return []gokugen.MiddlewareFunc{ts.storeTask}
}

// Sync syncs itnernal state with external data storage.
// Normally TaskStorage does it reversely through middlewares, mirroring internal state to external data storage.
// But after rebooting system, or repository is changed externally, Sync is needed to fetch back external data.
func (ts *SingleNodeTaskStorage) Sync(
	schedule func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
) (rescheduled map[string]gokugen.Task, schedulingErr map[string]error, err error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	syncedAt := ts.lastSynced
	fetchedIds, err := ts.repo.GetUpdatedSince(ts.lastSynced)
	if err != nil {
		return
	}

	rescheduled = make(map[string]gokugen.Task)
	schedulingErr = make(map[string]error)

	for _, fetched := range fetchedIds {
		if fetched.LastModified.After(syncedAt) {
			// Latest time of fetched tasks is next last-synced time.
			// We do want to set it correctly to avoid doubly syncing same entry.
			//
			// And also, GetUpdatedSince implemention may limit the number of fetched entries.
			// There could still be non-synced tasks that is modified at same time as syncedAt.
			syncedAt = fetched.LastModified
			// Also clear knownId for this time.
			// knowId storage is needed to avoid doubly syncing.
			// Racy clients may add or modify tasks for same second after this agent fetched lastly.
			//
			// Some of database support only one second presicion.
			// (e.g. strftime('%s') of sqlite3. you can also use `strftime('%s','now') || substr(strftime('%f','now'),4)`
			//   to enable milli second precision. But there could still be race conditions.)
			ts.knownIdForTime.Clear()
		}

		if ts.knownIdForTime.Has(fetched.Id) {
			continue
		} else {
			ts.knownIdForTime.Add(fetched.Id)
		}

		task, err := ts.sync(schedule, fetched)
		if err != nil {
			schedulingErr[fetched.Id] = err
		} else if task != nil {
			rescheduled[fetched.Id] = task
		}
	}

	if syncedAt.After(ts.lastSynced) {
		ts.lastSynced = syncedAt
	}
	return
}

func (ts *SingleNodeTaskStorage) sync(
	schedule func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
	fetched TaskInfo,
) (task gokugen.Task, schedulingErr error) {

	_, ok := ts.workRegistry.Load(fetched.WorkId)
	if !ok {
		return nil, fmt.Errorf("%w: unknown work id = %s", ErrNonexistentWorkId, fetched.WorkId)
	}

	switch fetched.State {
	case Working, Done, Cancelled, Failed:
		if inTaskMap, loaded := ts.taskMap.LoadAndDelete(fetched.Id); loaded {
			inTaskMap.CancelWithReason(ExternalStateChangeErr{
				id:    fetched.Id,
				state: fetched.State,
			})
		}
		return
	default:
		if ts.taskMap.Has(fetched.Id) {
			return
		}
	}

	if !ts.shouldRestore(fetched) {
		return
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

	task, err := schedule(ctx)
	if err != nil {
		return nil, err
	}
	ts.taskMap.LoadOrStore(fetched.Id, task)
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
