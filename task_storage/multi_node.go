package taskstorage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ngicks/gokugen"
)

var (
	ErrOtherNodeWorkingOnTheTask = errors.New("other node is already working on the task")
)

type RepositoryUpdater interface {
	Repository
	StateUpdater
}

// MultiNodeTaskStorage is almost same as SingleNodeTaskStorage but has one additional behavior.
// MultiNodeTaskStorage tries to mark tasks as Working state right before task is being worked on.
// If task is already marked, it fails to do task.
// Multiple nodes can be synced to same data storage through RepositoryUpdater interface
// And only one node will do task.
type MultiNodeTaskStorage struct {
	repo RepositoryUpdater
	sns  *SingleNodeTaskStorage
}

func NewMultiNodeTaskStorage(
	repo RepositoryUpdater,
	shouldRestore func(TaskInfo) bool,
	workRegistry WorkRegistry,
) *MultiNodeTaskStorage {
	return &MultiNodeTaskStorage{
		repo: repo,
		sns:  NewSingleNodeTaskStorage(repo, shouldRestore, workRegistry, nil),
	}
}

func (m *MultiNodeTaskStorage) markWorking(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		return handler(gokugen.WrapContext(ctx, gokugen.WithWorkFnWrapper(buildWrapper(m.repo))))
	}
}

func buildWrapper(repo RepositoryUpdater) gokugen.WorkFnWrapper {
	return func(self gokugen.SchedulerContext, workFn gokugen.WorkFn) gokugen.WorkFn {
		return func(ctx context.Context, scheduled time.Time) (any, error) {
			taskId, err := gokugen.GetTaskId(self)
			if err != nil {
				return nil, err
			}
			swapped, err := repo.UpdateState(taskId, Initialized, Working)
			if err != nil {
				return nil, err
			}
			if !swapped {
				return nil, fmt.Errorf("%w: task id = %s", ErrOtherNodeWorkingOnTheTask, taskId)
			}
			return workFn(ctx, scheduled)
		}
	}
}

func (m *MultiNodeTaskStorage) Middleware(freeParam bool) []gokugen.MiddlewareFunc {
	return append(m.sns.Middleware(freeParam), m.markWorking)
}

func (m *MultiNodeTaskStorage) Sync(
	schedule func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
) (rescheduled map[string]gokugen.Task, schedulingErr map[string]error, err error) {
	return m.sns.Sync(schedule)
}
