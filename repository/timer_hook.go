package repository

import (
	"sync"
	"time"

	"github.com/ngicks/gokugen/repository/gormtask"
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gommon/pkg/common"
)

// HookTimer hooks every method that would mutate Repository content, then updates the timer to next scheduled task.
type HookTimer interface {
	AddTask(param scheduler.TaskParam)
	Cancel(id string)
	MarkAsDispatched(id string)
	Update(id string, param scheduler.TaskParam)
	StartTimer()
	StopTimer()
	TimerChannel() <-chan time.Time
}

var _ HookTimer = &HookTimerImpl{}

type HookTimerImpl struct {
	mu             sync.RWMutex
	initialized    bool
	cachedMin      gormtask.GormTask
	core           GormCore
	isTimerStarted bool
	NowGetter      common.NowGetter
	Timer          common.Timer
}

func NewHookTimer(core GormCore) (*HookTimerImpl, error) {
	initialized := true
	next, err := core.GetNext()
	if err != nil {
		if scheduler.IsEmpty(err) {
			initialized = false
		} else {
			return nil, err
		}
	}

	return &HookTimerImpl{
		initialized: initialized,
		cachedMin:   next,
		core:        core,
		NowGetter:   common.NowGetterReal{},
		Timer:       common.NewTimerReal(),
	}, nil
}

func (t *HookTimerImpl) updateWithLock() error {
	if !t.isTimerStarted {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return t.update()
}

func (t *HookTimerImpl) update() error {
	if !t.Timer.Stop() {
		select {
		case <-t.Timer.C():
		default:
		}
	}

	next, err := t.core.GetNext()
	if err != nil {
		return err
	}
	t.initialized = true
	t.cachedMin = next

	t.Timer.Reset(next.ScheduledAt.Sub(t.NowGetter.GetNow()))

	return nil
}

func (t *HookTimerImpl) AddTask(param scheduler.TaskParam) {
	t.mu.RLock()
	if !t.initialized || param.ToTask(false).Less(t.cachedMin.ToTask()) {
		go t.mu.RUnlock()
		t.updateWithLock()
		return
	}
	t.mu.RUnlock()
}

func (t *HookTimerImpl) Cancel(id string) {
	t.mu.RLock()
	if t.cachedMin.Id == id {
		go t.mu.RUnlock()
		t.updateWithLock()
		return
	}
	t.mu.RUnlock()
}
func (t *HookTimerImpl) MarkAsDispatched(id string) {
	// usually dispatched element is scheduled item.
	t.mu.Lock()
	defer t.mu.Unlock()

	if id == t.cachedMin.Id {
		t.update()
	}
}
func (t *HookTimerImpl) Update(id string, param scheduler.TaskParam) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if id == t.cachedMin.Id {
		if param.Priority == nil && param.ScheduledAt.IsZero() {
			// it's just param update
			return
		}

		if param.Priority == nil {
			p := t.cachedMin.Priority
			param.Priority = &p
		}
		if param.ScheduledAt.IsZero() {
			param.ScheduledAt = t.cachedMin.ScheduledAt
		}
		if param.ToTask(false).Less(t.cachedMin.ToTask()) {
			t.update()
			return
		}
	}
	// 1) id is updated to be before
	updatedToBefore := !param.ScheduledAt.IsZero() && param.ScheduledAt.Before(t.cachedMin.ScheduledAt)
	if updatedToBefore {
		t.update()
		return
	}
	// 2) id is scheduled at the same time as cachedMin is, and priority is updated.
	updatedToHigherPriority := (param.Priority != nil && param.ScheduledAt.IsZero() && *param.Priority > t.cachedMin.Priority)
	if updatedToHigherPriority {
		t.update()
		return
	}
}

func (t *HookTimerImpl) StartTimer() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isTimerStarted = true
	t.update()
}

func (t *HookTimerImpl) StopTimer() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.Timer.Stop() {
		select {
		case <-t.Timer.C():
		default:
		}
	}
	t.isTimerStarted = false
}

func (t *HookTimerImpl) TimerChannel() <-chan time.Time {
	return t.Timer.C()
}
