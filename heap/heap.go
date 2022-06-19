package heap

import (
	typeparamcommon "github.com/ngicks/type-param-common"
)

type ExcludableHeap[T any] struct {
	*typeparamcommon.HeapWrapper[T]
	internal *typeparamcommon.SliceInterface[T]
}

func NewHeap[T any](less func(i, j T) bool) *ExcludableHeap[T] {
	if less == nil {
		return nil
	}
	heapInternal, interfaceInternal := typeparamcommon.MakeHeap(less)
	h := &ExcludableHeap[T]{
		HeapWrapper: heapInternal,
		internal:    interfaceInternal,
	}
	h.Init()
	return h
}

func (exh *ExcludableHeap[T]) Len() int {
	return exh.internal.Len()
}
func (exh *ExcludableHeap[T]) Exclude(filter func(ent T) bool, start, end int) (removed []T) {
	if filter == nil {
		return
	}

	if start < 0 {
		start = 0
	} else if start >= len(exh.internal.Inner) {
		return
	}
	if end > len(exh.internal.Inner) {
		end = len(exh.internal.Inner)
	}

	if start > end {
		return
	}

	for i := start; i < end; i++ {
		if filter(exh.internal.Inner[i]) {
			removed = append(removed, exh.internal.Inner[i])
			exh.internal.Inner = append(exh.internal.Inner[:i], exh.internal.Inner[i+1:]...)
			end--
			i--
		}
	}

	exh.Init()
	return removed
}

func (h *ExcludableHeap[T]) Peek() (p T) {
	if len(h.internal.Inner) == 0 {
		return
	}
	return h.internal.Inner[0]
}
