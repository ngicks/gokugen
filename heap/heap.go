package heap

import (
	"container/heap"
)

// Internal generic heap
type gHeap[T any] struct {
	hi *heapInterface[T]
}

func NewHeap[T any](less func(i, j T) bool) *gHeap[T] {
	if less == nil {
		return nil
	}
	h := &gHeap[T]{
		hi: NewInterface(less),
	}
	h.init()
	return h
}

func (h *gHeap[T]) init() {
	heap.Init(h.hi)
}

func (h *gHeap[T]) Len() int {
	return h.hi.Len()
}

func (h *gHeap[T]) Push(ele T) {
	heap.Push(h.hi, ele)
}

func (h *gHeap[T]) Pop() T {
	c, ok := heap.Pop(h.hi).(T)
	if !ok {
		panic("invariant violated")
	}
	return c
}

func (h *gHeap[T]) Remove(i int) T {
	c, ok := heap.Remove(h.hi, i).(T)
	if !ok {
		panic("invariant violated")
	}
	return c
}

func (h *gHeap[T]) Fix(i int) {
	heap.Fix(h.hi, i)
}

func (h *gHeap[T]) Peek() (p T) {
	c, ok := h.hi.Peek().(T)
	if !ok {
		return
	}
	return c
}

// Exclude excludes elements from underlying heap if filter returns true.
// Heap is scanned in given range, namely [start,end).
// If filter func is nil, Exclude is no-op.
// The complexity of execlusion is O(n) where n = end-start,
// and restoration of heap invariants is O(m) where m = new len.
func (s *gHeap[T]) Exclude(filter func(ent T) bool, start, end int) (removed []T) {
	r := s.hi.Exclude(func(ent any) bool {
		return filter(ent.(T))
	}, start, end)

	for i := range r {
		removed = append(removed, r[i].(T))
	}
	s.init()
	return
}
