package heap

func NewInterface[T any](less func(i, j T) bool) *heapInterface[T] {
	if less == nil {
		return nil
	}
	return &heapInterface[T]{
		s:    make([]T, 0),
		less: less,
	}
}

type heapInterface[T any] struct {
	s    []T
	less func(i, j T) bool
}

func (s *heapInterface[T]) Len() int {
	return len(s.s)
}

// Less means j is less than i
func (s *heapInterface[T]) Less(i, j int) bool {
	return s.less(s.s[i], s.s[j])
}

// Swap swaps the elements with indexes i and j.
func (s *heapInterface[T]) Swap(i, j int) {
	s.s[i], s.s[j] = s.s[j], s.s[i]
}

func (s *heapInterface[T]) Push(x any) {
	c, ok := x.(T)
	if !ok {
		panic("invariant violation")
	}
	s.s = append(s.s, c)
}

func (s *heapInterface[T]) Pop() (p any) {
	p, s.s = s.s[len(s.s)-1], s.s[:len(s.s)-1]
	return
}

func (s *heapInterface[T]) Peek() (p any) {
	if len(s.s) != 0 {
		return s.s[0]
	} else {
		return
	}
}

func (s *heapInterface[T]) Exclude(filter func(ent any) bool, start, end int) (removed []any) {
	if filter == nil {
		return
	}

	if start < 0 {
		start = 0
	} else if start >= len(s.s) {
		return
	}
	if end > len(s.s) {
		end = len(s.s)
	}

	if start > end {
		return
	}

	for i := start; i < end; i++ {
		if filter(s.s[i]) {
			removed = append(removed, s.s[i])
			s.s = append(s.s[:i], s.s[i+1:]...)
			end--
			i--
		}
	}
	return removed
}
