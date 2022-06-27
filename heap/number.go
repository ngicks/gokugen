package heap

import (
	typeparamcommon "github.com/ngicks/type-param-common"
	"golang.org/x/exp/constraints"
)

func NewNumber[T constraints.Ordered]() *ExcludableHeap[T] {
	heapInternal, interfaceInternal := typeparamcommon.MakeMinHeap[T]()
	h := &ExcludableHeap[T]{
		HeapWrapper: heapInternal,
		internal:    interfaceInternal,
	}
	h.Init()
	return h
}
