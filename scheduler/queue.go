package scheduler

import (
	"errors"
	"sync"

	"github.com/ngicks/gokugen/heap"
)

var (
	ErrMax = errors.New("heap size exceeds max limit")
)

type TaskQueue interface {
	// Len returns number of element stored in underlying heap.
	// The complexity of typical implementation is O(1)
	Len() int
	// Push pushes *Task into heap.
	// Push may return ErrMax if max is set, namely max != 0.
	// The complexity is O(log n) where n = t.Len()
	Push(task *Task) error
	// Peek peeks a min element in heap without any change.
	// The complexity of typical implementation is O(1)
	Peek() *Task
	// Pop pops a min element from heap if given `checker` function is nil or `checker` returns true.
	// `checker` will be called with a min element.
	// Argument to `checker` would be nil if heap is empty.
	// Pop returns nil when heap is empty, given `checker` function is nil or `checker` returns false.
	Pop(checker func(*Task) bool) *Task
	// Exclude excludes elements from underlying heap if filter returns true.
	// Heap is scanned in given range, namely [start,end).
	// Wider range is, longer it will hold lock. Range size must be chosen wisely.
	// If filter func is nil, it is no-op.
	// The complexity of execlusion is O(n) where n = end-start,
	// and restoration of heap invariants is O(m) where m = new len.
	// Number of exclued elements has significant performance effect much more than restoration of invariants.
	Exclude(filter func(ent *Task) bool, start, end int) (removed []*Task)
}

type internalHeap interface {
	Len() int
	Peek() *Task
	Pop() *Task
	Push(ele *Task)
	Remove(i int) *Task
	Fix(i int)
	Exclude(filter func(ent *Task) bool, start, end int) (removed []*Task)
}

func isUnlimited(u uint) bool {
	return u == 0
}

func less(i, j *Task) bool {
	return i.scheduledTime.Before(j.scheduledTime)
}

type SyncTaskQueue struct {
	mu   sync.Mutex
	max  uint
	heap internalHeap
}

func NewSyncQueue(max uint) *SyncTaskQueue {
	return &SyncTaskQueue{heap: heap.NewHeap(less), max: max}
}

func (q *SyncTaskQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.heap.Len()
}

func (q *SyncTaskQueue) Push(task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !isUnlimited(q.max) && uint(q.heap.Len()) >= q.max {
		return ErrMax
	}

	q.heap.Push(task)
	return nil
}

func (q *SyncTaskQueue) Peek() *Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.heap.Peek()
}

func (q *SyncTaskQueue) Pop(checker func(*Task) bool) *Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	if checker != nil && checker(q.heap.Peek()) {
		return q.heap.Pop()
	} else {
		return nil
	}
}

func (q *SyncTaskQueue) Exclude(filter func(ent *Task) bool, start, end int) (removed []*Task) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.heap.Exclude(filter, start, end)
}

type UnsafeTaskQueue struct {
	max  uint
	heap internalHeap
}

func NewUnsafeQueue(max uint) *UnsafeTaskQueue {
	return &UnsafeTaskQueue{heap: heap.NewHeap(less), max: max}
}

func (q *UnsafeTaskQueue) Push(task *Task) error {
	if !isUnlimited(q.max) && uint(q.heap.Len()) >= q.max {
		return ErrMax
	}
	q.heap.Push(task)
	return nil
}

func (q *UnsafeTaskQueue) Len() int {
	return q.heap.Len()
}

func (q *UnsafeTaskQueue) Peek() *Task {
	return q.heap.Peek()
}

func (q *UnsafeTaskQueue) Pop(checker func(*Task) bool) *Task {
	if checker == nil || checker(q.heap.Peek()) {
		return q.heap.Pop()
	} else {
		return nil
	}
}

func (q *UnsafeTaskQueue) Exclude(filter func(ent *Task) bool, start, end int) (removed []*Task) {
	return q.heap.Exclude(filter, start, end)
}
