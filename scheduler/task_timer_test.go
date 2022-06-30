package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/ngicks/gokugen/common"
	"github.com/ngicks/gokugen/scheduler"
)

func TestTaskTimer(t *testing.T) {
	t.Run("TaskTimer basic usage", func(t *testing.T) {
		now := time.Now()
		g := getNowDummyImpl{
			dummy: now,
		}

		tasks := make([]*scheduler.Task, 0, 100)
		for i := int64(0); i < 100; i++ {
			tasks = append(
				tasks,
				scheduler.NewTask(
					now.Add(time.Duration(i*int64(time.Microsecond))),
					func(taskCtx context.Context, scheduled time.Time) {},
				),
			)
		}

		f := scheduler.NewTaskTimer(0, &g, common.NewTimerImpl())

		var task []*scheduler.Task
		task = f.GetScheduledTask(now)
		if len(task) != 0 {
			t.Fatalf("not 0: %d", len(task))
		}

		for _, tt := range tasks {
			f.Push(tt)
		}

		task = f.GetScheduledTask(now)
		if len(task) != 1 {
			t.Fatalf("not 1: %d", len(task))
		}
		if task[0] != tasks[0] {
			t.Fatalf("not matching")
		}

		task = f.GetScheduledTask(now.Add(50 * time.Microsecond))
		if len(task) != 50 {
			t.Fatalf("not 50: %d", len(task))
		}
		if task[0] != tasks[1] {
			t.Fatalf("not matching")
		}
		if task[49] != tasks[50] {
			t.Fatalf("not matching")
		}

		task = f.GetScheduledTask(now.Add(1000 * time.Microsecond))
		if len(task) != 49 {
			t.Fatalf("not 49: %d", len(task))
		}

		f.Stop()
	})

	t.Run("timer reset", func(t *testing.T) {
		now := time.Now()
		g := getNowDummyImpl{
			dummy: now,
		}

		dummyTimer := func() *timerDummyImpl {
			timer := time.NewTimer(time.Nanosecond)
			if !timer.Stop() {
				<-timer.C
			}
			return &timerDummyImpl{timer: common.NewTimerImpl()}
		}()
		f := scheduler.NewTaskTimer(0, &g, dummyTimer)

		f.Push(scheduler.NewTask(
			now.Add(-2*time.Second),
			func(taskCtx context.Context, scheduled time.Time) {},
		))

		if d := dummyTimer.resetArg[0]; len(dummyTimer.resetArg) != 1 || d != -2*time.Second {
			t.Fatalf("reset duration is not as intended; must be %d, but is %d", -2*time.Second, d)
		}

		f.Push(scheduler.NewTask(
			now.Add(-time.Second),
			func(taskCtx context.Context, scheduled time.Time) {},
		))

		if len(dummyTimer.resetArg) != 1 {
			t.Fatalf("reset must not happen")
		}

		f.GetScheduledTask(now)

		f.Push(scheduler.NewTask(
			now.Add(5*time.Second),
			func(taskCtx context.Context, scheduled time.Time) {},
		))

		if d := dummyTimer.resetArg[1]; len(dummyTimer.resetArg) != 2 || d != 5*time.Second {
			t.Fatalf("reset duration is not as intended; must be %d, but is %d", 5*time.Second, d)
		}

		f.Push(scheduler.NewTask(
			now.Add(3*time.Second),
			func(taskCtx context.Context, scheduled time.Time) {},
		))

		if d := dummyTimer.resetArg[2]; len(dummyTimer.resetArg) != 3 || d != 3*time.Second {
			t.Fatalf("reset duration is not as intended; must be %d, but is %d", 3*time.Second, d)
		}

		f.GetScheduledTask(now.Add(3 * time.Second))

		if d := dummyTimer.resetArg[3]; len(dummyTimer.resetArg) != 4 || d != 5*time.Second {
			t.Fatalf("reset duration is not as intended; must be %d, but is %d", 5*time.Second, d)
		}

		f.GetScheduledTask(now.Add(5 * time.Second))

		if len(dummyTimer.resetArg) != 4 {
			t.Fatalf("reset must not happen")
		}
	})
}
