package heap

type Number interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

func less[T Number](i, j T) bool {
	return i < j
}

func NewNumber[T Number]() *gHeap[T] {
	return NewHeap(less[T])
}
