package repository

import (
	"testing"
	"time"

	"github.com/ngicks/mockable"
)

func TestMutationHookTimer(t *testing.T) {
	hook := NewMutationHookTimer()
	fakeTimer := mockable.NewClockFake(time.Now())
	hook.clock = fakeTimer
	testHookTimer(t, hook, fakeTimer)
}
