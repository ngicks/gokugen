package heap

import typeparamcommon "github.com/ngicks/type-param-common"

func NewNumber[T typeparamcommon.Lessable]() *ExcludableHeap[T] {
	heapInternal, interfaceInternal := typeparamcommon.MakeMinHeap[T]()
	h := &ExcludableHeap[T]{
		HeapWrapper: heapInternal,
		internal:    interfaceInternal,
	}
	h.Init()
	return h
}
