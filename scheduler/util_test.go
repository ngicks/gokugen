package scheduler_test

import (
	"time"

	"github.com/ngicks/gokugen/common"
)

var _ common.GetNow = new(getNowDummyImpl)
var _ common.ITimer = new(timerDummyImpl)

type getNowDummyImpl struct {
	dummy time.Time
}

func (g *getNowDummyImpl) GetNow() time.Time {
	return g.dummy
}

type timerDummyImpl struct {
	resetArg []time.Duration
	timer    *common.TimerImpl
}

func (t *timerDummyImpl) GetChan() <-chan time.Time {
	return t.timer.C
}

func (t *timerDummyImpl) Reset(to, now time.Time) {
	t.resetArg = append(t.resetArg, to.Sub(now))
	t.timer.Reset(to, now)
}
func (t *timerDummyImpl) Stop() {
	t.timer.Stop()
}
