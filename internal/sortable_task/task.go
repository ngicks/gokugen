package sortabletask

import (
	"sync/atomic"
	"time"

	"github.com/ngicks/generic/slice"
	"github.com/ngicks/genericcontainer/heapimpl"
	"github.com/ngicks/gokugen/def"
)

var _ setter = (*IndexedTask)(nil)

type IndexedTask struct {
	Task           *def.Task
	Index          int
	InsertionOrder uint64
}

func (t *IndexedTask) SetIndex(i int) {
	t.Index = i
}
func (t *IndexedTask) ScheduledAt() time.Time {
	return t.Task.ScheduledAt
}
func (t *IndexedTask) Priority() int {
	return t.Task.Priority
}
func (t *IndexedTask) CreatedAt() time.Time {
	return t.Task.CreatedAt
}
func (t *IndexedTask) GetInsertionOrder() uint64 {
	return t.InsertionOrder
}

func WrapTask(task def.Task, count *atomic.Uint64) *IndexedTask {
	return &IndexedTask{
		Task:           &task,
		InsertionOrder: count.Add(1),
	}
}

type setter interface {
	SetIndex(int)
	ScheduledAt() time.Time
	Priority() int
	CreatedAt() time.Time
	GetInsertionOrder() uint64
}

func MakeHeapMethodSet[T setter]() heapimpl.HeapMethods[T] {
	return heapimpl.HeapMethods[T]{
		Swap: swap[T],
		Push: push[T],
		Pop:  pop[T],
	}
}

func Less[T setter](i, j T) bool {
	if !i.ScheduledAt().Equal(j.ScheduledAt()) {
		return i.ScheduledAt().Before(j.ScheduledAt())
	}

	if i.Priority() != j.Priority() {
		return i.Priority() > j.Priority()
	}

	if !i.CreatedAt().Equal(j.CreatedAt()) {
		return i.CreatedAt().Before(j.CreatedAt())
	}
	return i.GetInsertionOrder() < j.GetInsertionOrder()
}

func swap[T setter](slice *slice.Stack[T], i, j int) {
	(*slice)[i], (*slice)[j] = (*slice)[j], (*slice)[i]
	(*slice)[i].SetIndex(i)
	(*slice)[j].SetIndex(j)
}

func push[T setter](slice *slice.Stack[T], v T) {
	v.SetIndex(slice.Len())
	slice.Push(v)
}

func pop[T setter](slice *slice.Stack[T]) T {
	popped, ok := slice.Pop()
	if !ok {
		// let it panic here, with out of range message.
		_ = (*(*[]T)(slice))[slice.Len()-1]
	}
	popped.SetIndex(-1)
	return popped
}
