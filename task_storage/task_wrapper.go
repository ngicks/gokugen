package taskstorage

import (
	"github.com/ngicks/gokugen"
)

var _ gokugen.Task = &taskWrapper{}

type taskWrapper struct {
	gokugen.Task
	cancel func(baseCanceller func(err error) (cancelled bool)) func(err error) (cancelled bool)
}

func (fw *taskWrapper) Cancel() (cancelled bool) {
	return fw.cancel(func(err error) (cancelled bool) { return fw.Task.Cancel() })(nil)
}

func (fw *taskWrapper) CancelWithReason(reason error) (cancelled bool) {
	return fw.cancel(fw.Task.CancelWithReason)(reason)
}
