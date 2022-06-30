package common

//go:generate mockgen -source timer.go -destination __mock/timer.go

import "time"

// ITimer is timer interface.
// Intention is to use as an unexported field of some structs.
// And make it mock-able inside internal tests.
type ITimer interface {
	GetChan() <-chan time.Time
	Reset(to, now time.Time)
	Stop()
}

// TimerImpl is a struct that implements ITimer.
type TimerImpl struct {
	*time.Timer
}

// NewTimerImpl returns newly created TimerImpl.
// Timer is stopped after return.
func NewTimerImpl() *TimerImpl {
	timer := time.NewTimer(time.Second)
	if !timer.Stop() {
		<-timer.C
	}
	return &TimerImpl{timer}
}

func (t *TimerImpl) GetChan() <-chan time.Time {
	return t.C
}

func (t *TimerImpl) Stop() {
	if !t.Timer.Stop() {
		// non-blocking receive.
		// in case of racy concurrent receivers.
		select {
		case <-t.C:
		default:
		}
	}
}

func (t *TimerImpl) Reset(to, now time.Time) {
	t.Stop()
	t.Timer.Reset(to.Sub(now))
}
