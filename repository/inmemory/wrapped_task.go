package inmemory

import (
	"github.com/ngicks/generic/slice"
	"github.com/ngicks/gokugen/def"
)

type indexedTask struct {
	task  *def.Task
	index int
}

func wrapTask(task def.Task) *indexedTask {
	return &indexedTask{
		task: &task,
	}
}

func (t *indexedTask) Less(i, j *indexedTask) bool {
	return i.task.Less(*j.task)
}

func (t *indexedTask) Swap(slice *slice.Stack[*indexedTask], i, j int) {
	(*slice)[i], (*slice)[j] = (*slice)[j], (*slice)[i]
	(*slice)[i].index = i
	(*slice)[j].index = j
}

func (t *indexedTask) Push(slice *slice.Stack[*indexedTask], v *indexedTask) {
	v.index = slice.Len()
	slice.Push(v)
}

func (t *indexedTask) Pop(slice *slice.Stack[*indexedTask]) *indexedTask {
	popped, ok := slice.Pop()
	if !ok {
		// let it panic here, with out of range message.
		_ = (*(*[]*indexedTask)(slice))[slice.Len()-1]
	}
	popped.index = -1
	return popped
}
