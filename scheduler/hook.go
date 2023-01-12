package scheduler

import (
	"sync"

	"github.com/ngicks/type-param-common/set"
)

type LoopHooks interface {
	OnGetNextError(err error) error
	OnGetNext(task Task)
	OnDispatchError(task Task, err error) error
	OnDispatch(task Task)
	OnUpdateError(task Task, updateType UpdateType, err error) error
	OnUpdate(task Task, updateType UpdateType)
	OnTaskDone(task Task, err error)
}

// PassThroughHook is the simplest implementation of LoopHooks.
// It does nothing; it only returns the passed error.
type PassThroughHook struct{}

func (h PassThroughHook) OnGetNextError(err error) error {
	return err
}
func (h PassThroughHook) OnGetNext(_ Task) {}
func (h PassThroughHook) OnDispatchError(_ Task, err error) error {
	return err
}
func (h PassThroughHook) OnDispatch(_ Task) {}
func (h PassThroughHook) OnUpdateError(_ Task, _ UpdateType, err error) error {
	return err
}
func (h PassThroughHook) OnUpdate(_ Task, _ UpdateType) {}
func (h PassThroughHook) OnTaskDone(_ Task, _ error)    {}
