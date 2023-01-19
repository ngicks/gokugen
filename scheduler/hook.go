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

type PartialHook struct {
	OnGetNextError  func(err error) error
	OnGetNext       func(task Task)
	OnDispatchError func(task Task, err error) error
	OnDispatch      func(task Task)
	OnUpdateError   func(task Task, updateType UpdateType, err error) error
	OnUpdate        func(task Task, updateType UpdateType)
	OnTaskDone      func(task Task, err error)
}

var _ LoopHooks = &ChainHook{}

// ChainHook is series of PartialHook-s.
// All of non-nil PartialHook methods are called with inputs in the normal iteration order of slice.
// If more than 2 PartialHooks implements the method that returns the error,
// input err is one that the previous hook returned.
//
// If no Chain element implements the method of the corresponding name, it falls back to the PassThroughHook.
type ChainHook struct {
	Chain           []PartialHook
	passThroughHook PassThroughHook
}

func (h *ChainHook) OnGetNextError(err error) error {
	hasImpl := false
	for _, hook := range h.Chain {
		if hook.OnGetNextError != nil {
			hasImpl = true
			err = hook.OnGetNextError(err)
		}
	}
	if hasImpl {
		return err
	}
	return h.passThroughHook.OnGetNextError(err)
}
func (h *ChainHook) OnGetNext(task Task) {
	for _, hook := range h.Chain {
		if hook.OnGetNext != nil {
			hook.OnGetNext(task)
		}
	}
}
func (h *ChainHook) OnDispatchError(task Task, err error) error {
	hasImpl := false
	for _, hook := range h.Chain {
		if hook.OnDispatchError != nil {
			hasImpl = true
			err = hook.OnDispatchError(task, err)
		}
	}
	if hasImpl {
		return err
	}
	return h.passThroughHook.OnDispatchError(task, err)
}
func (h *ChainHook) OnDispatch(task Task) {
	for _, hook := range h.Chain {
		if hook.OnDispatch != nil {
			hook.OnDispatch(task)
		}
	}
}
func (h *ChainHook) OnUpdateError(task Task, updateType UpdateType, err error) error {
	hasImpl := false
	for _, hook := range h.Chain {
		if hook.OnUpdateError != nil {
			hasImpl = true
			err = hook.OnUpdateError(task, updateType, err)
		}
	}
	if hasImpl {
		return err
	}
	return h.passThroughHook.OnUpdateError(task, updateType, err)
}
func (h *ChainHook) OnUpdate(task Task, updateType UpdateType) {
	for _, hook := range h.Chain {
		if hook.OnUpdate != nil {
			hook.OnUpdate(task, updateType)
		}
	}
}
func (h *ChainHook) OnTaskDone(task Task, err error) {
	for _, hook := range h.Chain {
		if hook.OnTaskDone != nil {
			hook.OnTaskDone(task, err)
		}
	}
}

type OnTaskDone = func(task Task, err error)

type hookWrapper struct {
	LoopHooks
	sync.RWMutex
	onTaskDone *set.OrderedSet[*OnTaskDone]
}

func newHookWrapper(hooks LoopHooks) *hookWrapper {
	return &hookWrapper{
		LoopHooks:  hooks,
		onTaskDone: set.NewOrdered[*OnTaskDone](),
	}
}

func (h *hookWrapper) addOnTaskDone(fn *OnTaskDone) {
	if fn == nil || *fn == nil {
		return
	}

	h.Lock()
	h.onTaskDone.Add(fn)
	h.Unlock()
}

func (h *hookWrapper) removeOnTaskDone(fn *OnTaskDone) {
	if fn == nil || *fn == nil {
		return
	}

	h.Lock()
	h.onTaskDone.Delete(fn)
	h.Unlock()
}

func (h *hookWrapper) OnTaskDone(task Task, err error) {
	h.LoopHooks.OnTaskDone(task, err)

	h.RLock()
	defer h.RUnlock()

	h.onTaskDone.ForEach(func(fn *OnTaskDone, _ int) {
		(*fn)(task, err)
	})
}
