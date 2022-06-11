package taskstorage

import (
	"fmt"
	"time"

	"github.com/ngicks/gokugen"
)

type RepositoryUpdater interface {
	Repository
	StateUpdater
}

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
		sns:  NewSingleNodeTaskStorage(repo, shouldRestore, workRegistry),
	}
}

func (m *MultiNodeTaskStorage) markWorking(handler gokugen.ScheduleHandlerFn) gokugen.ScheduleHandlerFn {
	return func(ctx gokugen.SchedulerContext) (gokugen.Task, error) {
		taskId, err := GetTaskId(ctx)
		if err != nil {
			return nil, err
		}

		return handler(
			&fnWrapperCtx{
				SchedulerContext: ctx,
				wrapper: func(_ gokugen.SchedulerContext, workFn WorkFn) WorkFn {
					return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
						swapped, err := m.repo.UpdateState(taskId, Initialized, Working)
						if err != nil {
							return err
						}
						if !swapped {
							return fmt.Errorf("%w: task id = %s", ErrOtherNodeWorkingOnTheTask, taskId)
						}
						return workFn(ctxCancelCh, taskCancelCh, scheduled)
					}
				},
			})
	}
}

func (m *MultiNodeTaskStorage) Middleware(freeParam bool) []gokugen.MiddlewareFunc {
	return append(m.sns.Middleware(freeParam), m.markWorking)
}

func (m *MultiNodeTaskStorage) Sync(
	schedule func(ctx gokugen.SchedulerContext) (gokugen.Task, error),
) (restored bool, rescheduled map[string]gokugen.Task, err error) {
	return m.sns.Sync(schedule)
}
