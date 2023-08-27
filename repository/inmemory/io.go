package inmemory

import (
	"fmt"

	"github.com/ngicks/gokugen/def"
	sortabletask "github.com/ngicks/gokugen/internal/sortable_task"
)

type KeyValue struct {
	Key   string
	Value def.Task
}

func (r *InMemoryRepository) Save() []KeyValue {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]KeyValue, 0, r.orderedMap.Len())
	for pair := r.orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		out = append(out, KeyValue{Key: pair.Key, Value: pair.Value.Task.Clone()})
	}
	return out
}

func (r *InMemoryRepository) Load(kv []KeyValue) error {
	for _, pair := range kv {
		if !pair.Value.IsValid() {
			return fmt.Errorf(
				"%w: task is invalid because = %+v",
				def.ErrInvalidTask, pair.Value.ReportInvalidity(),
			)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.init()
	for _, pair := range kv {
		wrapped := sortabletask.WrapTask(pair.Value.Clone(), r.insertionOrderCount)
		if pair.Value.State == def.TaskScheduled {
			r.heap.Push(wrapped)
		}
		r.orderedMap.Set(pair.Key, wrapped)
	}
	return nil
}
