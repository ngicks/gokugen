package cron

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/ngicks/genericcontainer/heapimpl"
	"github.com/ngicks/gokugen/def"
	sortabletask "github.com/ngicks/gokugen/internal/sortable_task"
	"github.com/ngicks/gokugen/mutator"
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/mockable"
)

var _ scheduler.VolatileTask = (*CronStore)(nil)

type CronStore struct {
	mu sync.Mutex

	insertionOrderCount *atomic.Uint64
	schedule            *heapimpl.FilterableHeap[*wrappedTask]
	mutators            mutator.MutatorStore
	entries             map[serializable]*Entry
	clock               mockable.Clock
}

func NewCronTable(entries []*Entry) (*CronStore, error) {
	c := &CronStore{
		insertionOrderCount: new(atomic.Uint64),
		schedule: heapimpl.NewFilterableHeapHooks[*wrappedTask](
			sortabletask.Less[*wrappedTask],
			sortabletask.MakeHeapMethodSet[*wrappedTask](),
		),
		mutators: mutator.DefaultMutatorStore,
		entries:  make(map[serializable]*Entry),
		clock:    mockable.NewClockReal(),
	}

	for _, ent := range entries {
		next := ent.Next()
		key := paramToSerializable(next)
		c.entries[key] = ent
		mutators, err := c.mutators.Load(next.Meta.Value())
		if err != nil {
			return nil, err
		}
		next = mutators.Apply(next)
		c.schedule.Push(
			&wrappedTask{
				key:      key,
				mutators: mutators,
				IndexedTask: sortabletask.WrapTask(
					next.ToTask(uuid.NewString(), c.clock.Now()),
					c.insertionOrderCount,
				),
			},
		)

	}
	return c, nil
}

func (c *CronStore) GetNext(ctx context.Context) (def.Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.schedule.Len() == 0 {
		return def.Task{}, &def.RepositoryError{Kind: def.Exhausted}
	}
	return c.schedule.Peek().Task.Clone(), nil
}

func (c *CronStore) Pop(ctx context.Context) (def.Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.schedule.Len() == 0 {
		return def.Task{}, &def.RepositoryError{Kind: def.Exhausted}
	}
	t := c.schedule.Pop()
	c.pushNext(t)
	c.resetTimer()
	return *t.Task, nil
}

func (c *CronStore) pushNext(t *wrappedTask) {
	ent := c.entries[t.key]
	next := ent.Next()
	next = t.mutators.Apply(next)
	c.schedule.Push(&wrappedTask{
		key:      t.key,
		mutators: t.mutators,
		IndexedTask: sortabletask.WrapTask(
			next.ToTask(uuid.NewString(), c.clock.Now()),
			c.insertionOrderCount,
		),
	})
}

func (c *CronStore) resetTimer() {
	if !c.clock.Stop() {
		select {
		case <-c.clock.C():
		default:
		}
	}

	if c.schedule.Len() > 0 {
		c.clock.Reset(c.schedule.Peek().Task.ScheduledAt.Sub(c.clock.Now()))
	}
}

func (c *CronStore) LastTimerUpdateError() error {
	return nil
}

func (c *CronStore) StartTimer(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.resetTimer()
}

func (c *CronStore) StopTimer() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.clock.Stop() {
		select {
		case <-c.clock.C():
		default:
		}
	}
}

func (c *CronStore) TimerChannel() <-chan time.Time {
	return c.clock.C()
}
