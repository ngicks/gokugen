package mock_gokugen

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	scheduler "github.com/ngicks/gokugen/scheduler"
)

func BuildMockScheduler(t *testing.T) (ctrl *gomock.Controller, mockSched *MockScheduler, getTrappedTask func() *scheduler.Task) {
	ctrl = gomock.NewController(t)
	mockSched = NewMockScheduler(ctrl)

	var trappedTask *scheduler.Task
	getTrappedTask = func() *scheduler.Task {
		return trappedTask
	}
	mockSched.
		EXPECT().
		Schedule(gomock.Any()).
		DoAndReturn(func(task *scheduler.Task) (*scheduler.TaskController, error) {
			trappedTask = task
			return scheduler.NewTaskController(task), nil
		}).
		AnyTimes()

	return
}
