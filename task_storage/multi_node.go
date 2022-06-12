package taskstorage

import (
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
		return handler(
			gokugen.WithWorkFnWrapper(
				ctx,
				func(self gokugen.SchedulerContext, workFn WorkFn) WorkFn {
					return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) error {
						taskId, err := gokugen.GetTaskId(self)
						if err != nil {
							return err
						}
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
			),
		)
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
