package common

import "time"

type ITimer interface {
	GetChan() <-chan time.Time
	Reset(to, now time.Time)
	Stop()
}

type TimerImpl struct {
	*time.Timer
}

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
