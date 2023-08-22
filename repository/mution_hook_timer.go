package repository

import (
	"context"
	"sync"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/mockable"
	"github.com/ngicks/und/option"
)

var _ HookTimer = &MutationHookTimer{}

type MutationHookTimer struct {
	mu             sync.RWMutex
	cachedMin      def.Task
	repo           def.Repository
	isTimerStarted bool
	lastErr        error
	clock          mockable.Clock
}

func NewMutationHookTimer() *MutationHookTimer {
	t := &MutationHookTimer{
		clock: mockable.NewClockReal(),
	}
	return t
}

func (t *MutationHookTimer) SetRepository(core def.Repository) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.repo != nil {
		panic("SetRepository is called twice")
	}

	t.repo = core
}

func (t *MutationHookTimer) LastTimerUpdateError() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastErr
}

func (t *MutationHookTimer) update(ctx context.Context) {
	t.lastErr = t._update(ctx)
}

func (t *MutationHookTimer) _update(ctx context.Context) error {
	if !t.isTimerStarted {
		return nil
	}

	if !t.clock.Stop() {
		select {
		case <-t.clock.C():
		default:
		}
	}

	next, err := t.repo.GetNext(ctx)
	// resets to zero-value if err.
	t.cachedMin = next

	if err == nil {
		t.clock.Reset(next.ScheduledAt.Sub(t.clock.Now()))
		return nil
	}

	if def.IsExhausted(err) {
		return nil
	}
	return err
}

// 30 years after
var farFuture = time.Now().Add(30 * 365 * 24 * time.Hour)

func (t *MutationHookTimer) AddTask(ctx context.Context, param def.TaskUpdateParam) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cachedMin.Id == "" || param.ToTask(def.NeverExistentId, farFuture).Less(t.cachedMin) {
		t.update(ctx)
	}
}

func (t *MutationHookTimer) UpdateById(ctx context.Context, id string, param def.TaskUpdateParam) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cachedMin.Id == "" {
		t.update(ctx)
		return
	}

	if id == t.cachedMin.Id {
		if param.Priority.IsNone() && param.ScheduledAt.IsNone() {
			// it's just param update
			return
		}

		param.Priority = param.Priority.Or(option.Some(t.cachedMin.Priority))
		param.ScheduledAt = param.ScheduledAt.Or(option.Some(t.cachedMin.ScheduledAt))

		if param.ToTask(def.NeverExistentId, time.Time{}).Less(t.cachedMin) {
			t.update(ctx)
			return
		}
	}

	// 1) id is updated to be before
	updatedToBefore := param.ScheduledAt.IsSome() &&
		param.ScheduledAt.Value().Before(t.cachedMin.ScheduledAt)
	if updatedToBefore {
		t.update(ctx)
		return
	}
	// 2) id is scheduled at the same time as cachedMin is, and priority is updated.
	updatedToHigherPriority := param.Priority.IsSome() &&
		(param.ScheduledAt.IsNone()) &&
		param.Priority.Value() > t.cachedMin.Priority
	if updatedToHigherPriority {
		t.update(ctx)
		return
	}
}

func (t *MutationHookTimer) Cancel(ctx context.Context, id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cachedMin.Id == "" || t.cachedMin.Id == id {
		t.update(ctx)
	}
}

func (t *MutationHookTimer) MarkAsDispatched(ctx context.Context, id string) {
	// usually dispatched element is scheduled item.
	t.mu.Lock()
	defer t.mu.Unlock()

	if id == t.cachedMin.Id {
		t.update(ctx)
	}
}

func (t *MutationHookTimer) StartTimer(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isTimerStarted = true
	t.update(ctx)
}

func (t *MutationHookTimer) StopTimer() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.clock.Stop() {
		select {
		case <-t.clock.C():
		default:
		}
	}
	t.isTimerStarted = false
}

func (t *MutationHookTimer) TimerChannel() <-chan time.Time {
	return t.clock.C()
}
