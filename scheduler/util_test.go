package scheduler_test

import (
	"time"

	"github.com/ngicks/gokugen/common"
	"github.com/ngicks/gokugen/scheduler"
)

var _ common.GetNow = new(getNowDummyImpl)
var _ scheduler.ITimer = new(timerDummyImpl)

type getNowDummyImpl struct {
	dummy time.Time
}

func (g *getNowDummyImpl) GetNow() time.Time {
	return g.dummy
}

type timerDummyImpl struct {
	resetArg []time.Duration
	timer    *time.Timer
}

func (t *timerDummyImpl) GetChan() <-chan time.Time {
	return t.timer.C
}

func (t *timerDummyImpl) Reset(d time.Duration) bool {
	t.resetArg = append(t.resetArg, d)
	return t.timer.Reset(d)
}
func (t *timerDummyImpl) Stop() bool {
	return t.timer.Stop()
}
