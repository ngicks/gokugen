package container

import (
	"sync"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	tpc "github.com/ngicks/type-param-common"
)

var _ TaskContainer = &HeapTaskContainer{}

type HeapTaskContainer struct {
	mu      sync.Mutex
	taskMap map[string]*TaskModel
	heap    HeapDesync
}

func NewHeapTaskContainer() *HeapTaskContainer {
	return &HeapTaskContainer{
		heap: *NewHeapDesync(),
	}
}

func (h *HeapTaskContainer) Cancel(id string) (cancelled bool, err error) {
	t, ok := h.taskMap[id]
	if ok {
		cancelled = t.setCancelled()
	}
	return cancelled, nil
}

func (h *HeapTaskContainer) AddTask(taskId string, param []byte, scheduledTime time.Time) (scheduler.Task, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	t, err := h.heap.AddTask(taskId, param, scheduledTime)
	if err != nil {
		return nil, err
	}
	h.taskMap[t.Id()] = t.(*TaskModel)
	return t, nil
}

func (h *HeapTaskContainer) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.heap.Len()
}

func (h *HeapTaskContainer) Peek() scheduler.Task {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.heap.Peek()
}

func (h *HeapTaskContainer) Pop() scheduler.Task {
	h.mu.Lock()
	defer h.mu.Unlock()
	t := h.heap.Pop()
	delete(h.taskMap, t.Id())
	return t
}

func (h *HeapTaskContainer) Exclude(start, end int, filter func(ent scheduler.Task) bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.heap.Exclude(start, end, func(ent scheduler.Task) bool {
		cond := filter(ent)
		if cond {
			delete(h.taskMap, ent.Id())
		}
		return cond
	})
}

type HeapDesync struct {
	heap tpc.FilterableHeap[*TaskModel, *TaskModel]
}

func NewHeapDesync() *HeapDesync {
	return &HeapDesync{
		heap: *tpc.NewFilterableHeap[*TaskModel, *TaskModel](),
	}
}

func (h *HeapDesync) AddTask(taskId string, param []byte, scheduledTime time.Time) (scheduler.Task, error) {
	t := NewTask(taskId, param, scheduledTime)
	h.heap.Push(t)
	return t, nil
}

// Len returns number of elements stored in underlying heap.
// The complexity of typical implementation is O(1)
func (h *HeapDesync) Len() int {
	return h.heap.Len()
}

// Peek peeks a min element in heap without changing it.
// The complexity of typical implementation is O(1)
func (h *HeapDesync) Peek() scheduler.Task {
	return h.heap.Peek()
}

// Pop pops a min element from heap if given `checker` function is nil or `checker` returns true.
func (h *HeapDesync) Pop() scheduler.Task {
	return h.heap.Pop()
}

func (h *HeapDesync) Exclude(start, end int, filter func(ent scheduler.Task) bool) {
	h.heap.Filter(
		tpc.BuildExcludeFilter(
			start,
			end,
			func(ent *TaskModel) bool {
				return filter(ent)
			},
		),
	)
}
