package taskstorage

import (
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
		return handler(newtaskStorageMarkingWorkingCtx(ctx, taskId, m.repo))
	}
}
