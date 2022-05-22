package scheduler

import "sync/atomic"

// workingState is togglable wroking state.
type workingState struct {
	working uint32
}

func (s *workingState) setWorking(to ...bool) (swapped bool) {
	setTo := true
	for _, setState := range to {
		if !setState {
			setTo = false
		}
	}
	if setTo {
		return atomic.CompareAndSwapUint32(&s.working, 0, 1)
	} else {
		return atomic.CompareAndSwapUint32(&s.working, 1, 0)
	}
}

func (s *workingState) IsWorking() bool {
	return atomic.LoadUint32(&s.working) == 1
}

// endState is reprensentation of one-way only transition to end state.
type endState struct {
	ended uint32
}

func (s *endState) setEnded() (swapped bool) {
	return atomic.CompareAndSwapUint32(&s.ended, 0, 1)
}

func (s *endState) IsEnded() bool {
	return atomic.LoadUint32(&s.ended) == 1
}
