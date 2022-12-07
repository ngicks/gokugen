package container

import (
	"errors"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

var ErrMax = errors.New("max")

type TaskContainer interface {
	TaskQueue
	AddTask(taskId string, param []byte, scheduledTime time.Time) (scheduler.Task, error)
	Cancel(id string) (cancelled bool, err error)
}

// TaskQueue is priority queue.
// Task with least scheduled time has most priority.
type TaskQueue interface {
	ConsumableQueue
	MutableQueue
}

type ConsumableQueue interface {
	// Len returns number of elements stored in underlying heap.
	// The complexity of typical implementation is O(1)
	Len() int
	// Peek peeks a min element in heap without changing it.
	// The complexity of typical implementation is O(1)
	Peek() scheduler.Task
	// Pop pops a min element from heap if given `checker` function is nil or `checker` returns true.
	Pop() scheduler.Task
}

type MutableQueue interface {
	// Exclude excludes elements from underlying heap if filter returns true.
	// Heap is scanned in given range, namely [start,end).
	// Wider range is, longer it will hold lock. Range size must be chosen wisely.
	// If filter func is nil, it is no-op.
	// The complexity of exclusion is O(n) where n = end-start,
	// and restoration of heap invariants is O(m) where m = new len.
	// Number of excluded elements has significant performance effect much more than restoration of invariants.
	Exclude(start, end int, filter func(ent scheduler.Task) bool)
}
