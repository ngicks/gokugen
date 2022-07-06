package scheduler_test

import (
	"time"

	"github.com/ngicks/gommon"
)

var _ gommon.GetNower = new(getNowDummyImpl)
var _ gommon.ITimer = new(timerDummyImpl)

type getNowDummyImpl struct {
	dummy time.Time
}

func (g *getNowDummyImpl) GetNow() time.Time {
	return g.dummy
}

type timerDummyImpl struct {
	resetArg []time.Duration
	timer    *gommon.TimerImpl
}

func (t *timerDummyImpl) Channel() <-chan time.Time {
	return t.timer.C
}

func (t *timerDummyImpl) Reset(to, now time.Time) {
	t.resetArg = append(t.resetArg, to.Sub(now))
	t.timer.Reset(to, now)
}
func (t *timerDummyImpl) Stop() {
	t.timer.Stop()
}
