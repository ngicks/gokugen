package cron

import (
	"context"
	"fmt"
	"slices"
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

func NewCronStore(entries []*Entry) (*CronStore, error) {
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

	if err := c.updateTask(entries, nil); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *CronStore) updateTask(added, removed []*Entry) error {
	type set struct {
		entry       *Entry
		wrappedTask *wrappedTask
	}

	entryMap := make(map[serializable]*set)

	removedKeys := make(map[serializable]struct{})
	for _, ent := range removed {
		removedKeys[paramToSerializable(ent.Param())] = struct{}{}
	}

	for _, ent := range added {
		next := ent.Next()
		key := paramToSerializable(next)

		_, cHas := c.entries[key]
		_, addedHas := entryMap[key]
		_, willRemove := removedKeys[key]
		if !willRemove && (cHas || addedHas) {
			return fmt.Errorf(
				"given entry is serialized to the value"+
					" which overlaps to existing *Entry, serialized to %#v",
				key,
			)
		}

		mutators, err := c.mutators.Load(next.Meta.Value())
		if err != nil {
			return err
		}
		next = mutators.Apply(next)
		wrapped := &wrappedTask{
			key:      key,
			mutators: mutators,
			IndexedTask: sortabletask.WrapTask(
				next.ToTask(uuid.NewString(), c.clock.Now()),
				c.insertionOrderCount,
			),
		}

		entryMap[key] = &set{
			entry:       ent,
			wrappedTask: wrapped,
		}
	}

	if len(removed) > 0 {
		for _, ent := range removed {
			key := paramToSerializable(ent.Param())
			delete(c.entries, key)
		}
		c.schedule.Filter(func(innerSlice *[]*wrappedTask) {
			*innerSlice = slices.DeleteFunc(*innerSlice, func(wt *wrappedTask) bool {
				for _, ent := range removed {
					key := paramToSerializable(ent.Param())
					if wt.key == key {
						return true
					}
				}
				return false
			})
		})
	}

	for key, set := range entryMap {
		c.entries[key] = set.entry
		c.schedule.Push(set.wrappedTask)
	}

	return nil
}

func (c *CronStore) EditTask(fn func(entries []*Entry) []*Entry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stopTimer()
	defer c.resetTimer()

	var entries []*Entry
	for _, ent := range c.entries {
		entries = append(entries, ent)
	}

	modified := fn(slices.Clone(entries))

	var added, removed []*Entry
	for _, modifiedEnt := range modified {
		if !slices.Contains(entries, modifiedEnt) {
			added = append(added, modifiedEnt)
		}
	}
	for _, ent := range entries {
		if !slices.Contains(modified, ent) {
			removed = append(removed, ent)
		}
	}

	return c.updateTask(added, removed)
}

func (c *CronStore) Schedule() []def.Task {
	c.mu.Lock()
	defer c.mu.Unlock()
	cloned := c.schedule.Clone()

	var out []def.Task
	for cloned.Len() > 0 {
		out = append(out, *cloned.Pop().IndexedTask.Task)
	}
	return out
}

func (c *CronStore) Peek(ctx context.Context) (def.Task, error) {
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
	c.stopTimer()
}

func (c *CronStore) stopTimer() {
	if !c.clock.Stop() {
		select {
		case <-c.clock.C():
		default:
		}
	}
}

func (c *CronStore) NextScheduled() (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.schedule.Len() > 0 {
		return c.schedule.Peek().Task.ScheduledAt, true
	} else {
		return time.Time{}, false
	}
}

func (c *CronStore) TimerChannel() <-chan time.Time {
	return c.clock.C()
}
