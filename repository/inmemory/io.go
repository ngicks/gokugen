package inmemory

import (
	"fmt"

	"github.com/ngicks/genericcontainer/heapimpl"
	"github.com/ngicks/gokugen/def"
	orderedmap "github.com/wk8/go-ordered-map/v2"
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
		out = append(out, KeyValue{Key: pair.Key, Value: *pair.Value.task})
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

	r.heap = heapimpl.NewFilterableHeap[*indexedTask]()
	r.orderedMap = orderedmap.New[string, *indexedTask]()
	for _, pair := range kv {
		wrapped := wrapTask(pair.Value)
		if pair.Value.State == def.TaskScheduled {
			r.heap.Push(wrapped)
		}
		r.orderedMap.Set(pair.Key, wrapped)
	}
	return nil
}
