package repository

import (
	"sync"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gommon/pkg/common"
)

// HookTimer watches repository mutation and resets its timer if necessary.
type HookTimer interface {
	// SetRepository sets Repository. Calling this twice may cause a runtime panic.
	SetRepository(core scheduler.RepositoryLike)
	AddTask(param scheduler.TaskParam)
	Cancel(id string)
	MarkAsDispatched(id string)
	Update(id string, param scheduler.TaskParam)
	scheduler.TimerLike
}

var _ HookTimer = &RepositoryTimer{}

type RepositoryTimer struct {
	mu             sync.RWMutex
	cachedMin      scheduler.Task
	core           scheduler.RepositoryLike
	isTimerStarted bool
	NowGetter      common.NowGetter
	Timer          common.Timer
}

func NewHookTimer() *RepositoryTimer {
	return &RepositoryTimer{
		NowGetter: common.NowGetterReal{},
		Timer:     common.NewTimerReal(),
	}
}

func (t *RepositoryTimer) SetRepository(core scheduler.RepositoryLike) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.core != nil {
		panic("SetRepository is called twice")
	}

	t.core = core
}

func (t *RepositoryTimer) updateWithLock() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.update()
}

func (t *RepositoryTimer) update() error {
	if !t.isTimerStarted {
		return nil
	}

	if !t.Timer.Stop() {
		select {
		case <-t.Timer.C():
		default:
		}
	}

	next, err := t.core.GetNext()
	// resets to zero-value if err.
	t.cachedMin = next

	if err != nil {
		return err
	}

	t.Timer.Reset(next.ScheduledAt.Sub(t.NowGetter.GetNow()))

	return nil
}

func (t *RepositoryTimer) AddTask(param scheduler.TaskParam) {
	t.mu.RLock()
	if t.cachedMin.Id == "" || param.ToTask(false).Less(t.cachedMin) {
		t.mu.RUnlock()
		t.updateWithLock()
		return
	}
	t.mu.RUnlock()
}

func (t *RepositoryTimer) Cancel(id string) {
	t.mu.RLock()
	if t.cachedMin.Id == "" || t.cachedMin.Id == id {
		t.mu.RUnlock()
		t.updateWithLock()
		return
	}
	t.mu.RUnlock()
}

func (t *RepositoryTimer) MarkAsDispatched(id string) {
	// usually dispatched element is scheduled item.
	t.mu.Lock()
	defer t.mu.Unlock()

	if id == t.cachedMin.Id {
		t.update()
	}
}
func (t *RepositoryTimer) Update(id string, param scheduler.TaskParam) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cachedMin.Id == "" {
		t.update()
		return
	}

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
		param.Param = nil // avoid buf clone.
		if param.ToTask(false).Less(t.cachedMin) {
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

func (t *RepositoryTimer) StartTimer() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isTimerStarted = true
	t.update()
}

func (t *RepositoryTimer) StopTimer() {
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

func (t *RepositoryTimer) TimerChannel() <-chan time.Time {
	return t.Timer.C()
}
