package inmemory

import (
	"sync/atomic"

	"github.com/ngicks/generic/slice"
	"github.com/ngicks/gokugen/def"
)

type indexedTask struct {
	task           *def.Task
	index          int
	insertionOrder uint64
}

func wrapTask(task def.Task, count *atomic.Uint64) *indexedTask {
	return &indexedTask{
		task:           &task,
		insertionOrder: count.Add(1),
	}
}

func (*indexedTask) Less(i, j *indexedTask) bool {
	if !i.task.ScheduledAt.Equal(j.task.ScheduledAt) {
		return i.task.ScheduledAt.Before(j.task.ScheduledAt)
	}

	if i.task.Priority != j.task.Priority {
		return i.task.Priority > j.task.Priority
	}

	if !i.task.CreatedAt.Equal(j.task.CreatedAt) {
		return i.task.CreatedAt.Before(j.task.CreatedAt)
	}
	return i.insertionOrder < j.insertionOrder
}

func (*indexedTask) Swap(slice *slice.Stack[*indexedTask], i, j int) {
	(*slice)[i], (*slice)[j] = (*slice)[j], (*slice)[i]
	(*slice)[i].index = i
	(*slice)[j].index = j
}

func (*indexedTask) Push(slice *slice.Stack[*indexedTask], v *indexedTask) {
	v.index = slice.Len()
	slice.Push(v)
}

func (*indexedTask) Pop(slice *slice.Stack[*indexedTask]) *indexedTask {
	popped, ok := slice.Pop()
	if !ok {
		// let it panic here, with out of range message.
		_ = (*(*[]*indexedTask)(slice))[slice.Len()-1]
	}
	popped.index = -1
	return popped
}
