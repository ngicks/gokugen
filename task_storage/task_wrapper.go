package taskstorage

import (
	"time"

	"github.com/ngicks/gokugen"
)

var _ gokugen.Task = &taskWrapper{}

type taskWrapper struct {
	base   gokugen.Task
	cancel func(baseCanceller func() (cancelled bool)) func() (cancelled bool)
}

func (fw *taskWrapper) Cancel() (cancelled bool) {
	return fw.cancel(fw.base.Cancel)()
}
func (fw *taskWrapper) GetScheduledTime() time.Time {
	return fw.base.GetScheduledTime()
}
func (fw *taskWrapper) IsCancelled() bool {
	return fw.base.IsCancelled()
}
func (fw *taskWrapper) IsDone() bool {
	return fw.base.IsDone()
}
