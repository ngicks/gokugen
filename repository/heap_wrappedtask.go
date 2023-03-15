package repository

import (
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/type-param-common/slice"
)

// wrappedTask is scheduler.Task with Index of heap slice and associated heap functions.
type wrappedTask struct {
	scheduler.Task
	Index int
}

func (t *wrappedTask) Less(i, j *wrappedTask) bool {
	return i.Task.Less(j.Task)
}

func (t *wrappedTask) Swap(slice *slice.Stack[*wrappedTask], i, j int) {
	(*slice)[i], (*slice)[j] = (*slice)[j], (*slice)[i]
	(*slice)[i].Index = i
	(*slice)[j].Index = j
}

func (t *wrappedTask) Push(slice *slice.Stack[*wrappedTask], v *wrappedTask) {
	v.Index = slice.Len()
	slice.Push(v)
}

func (t *wrappedTask) Pop(slice *slice.Stack[*wrappedTask]) *wrappedTask {
	popped, ok := slice.Pop()
	if !ok {
		// let it panic here, with out of range message.
		_ = (*(*[]*wrappedTask)(slice))[slice.Len()-1]
	}
	popped.Index = -1
	return popped
}
