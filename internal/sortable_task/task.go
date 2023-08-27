package sortabletask

import (
	"sync/atomic"

	"github.com/ngicks/generic/slice"
	"github.com/ngicks/gokugen/def"
)

type IndexedTask struct {
	Task           *def.Task
	Index          int
	InsertionOrder uint64
}

func WrapTask(task def.Task, count *atomic.Uint64) *IndexedTask {
	return &IndexedTask{
		Task:           &task,
		InsertionOrder: count.Add(1),
	}
}

func (*IndexedTask) Less(i, j *IndexedTask) bool {
	if !i.Task.ScheduledAt.Equal(j.Task.ScheduledAt) {
		return i.Task.ScheduledAt.Before(j.Task.ScheduledAt)
	}

	if i.Task.Priority != j.Task.Priority {
		return i.Task.Priority > j.Task.Priority
	}

	if !i.Task.CreatedAt.Equal(j.Task.CreatedAt) {
		return i.Task.CreatedAt.Before(j.Task.CreatedAt)
	}
	return i.InsertionOrder < j.InsertionOrder
}

func (*IndexedTask) Swap(slice *slice.Stack[*IndexedTask], i, j int) {
	(*slice)[i], (*slice)[j] = (*slice)[j], (*slice)[i]
	(*slice)[i].Index = i
	(*slice)[j].Index = j
}

func (*IndexedTask) Push(slice *slice.Stack[*IndexedTask], v *IndexedTask) {
	v.Index = slice.Len()
	slice.Push(v)
}

func (*IndexedTask) Pop(slice *slice.Stack[*IndexedTask]) *IndexedTask {
	popped, ok := slice.Pop()
	if !ok {
		// let it panic here, with out of range message.
		_ = (*(*[]*IndexedTask)(slice))[slice.Len()-1]
	}
	popped.Index = -1
	return popped
}
