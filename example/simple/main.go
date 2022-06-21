package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

func main() {
	if err := _main(); err != nil {
		panic(err)
	}
}

func printNowWithDelay(id int, delay time.Duration) func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) {
	return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) {
		now := time.Now()
		var isCtxCancelled, isTaskCancelled bool
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctxCancelCh:
			case <-taskCancelCh:
			}
		}
		select {
		case <-ctxCancelCh:
			isCtxCancelled = true
		default:
		}
		select {
		case <-taskCancelCh:
			isTaskCancelled = true
		default:
		}

		fmt.Printf(
			"id: %d, scheduled: %s, diff to now: %s, isCtxCancelled: %t, isTaskCancelled: %t\n",
			id,
			scheduled.Format(time.RFC3339Nano),
			now.Sub(scheduled).String(),
			isCtxCancelled,
			isTaskCancelled,
		)
	}
}

func _main() error {
	sched := scheduler.NewScheduler(5, 0)

	now := time.Now()
	printNow := func(id int) func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time) {
		return printNowWithDelay(id, 0)
	}

	sched.Schedule(scheduler.NewTask(now, printNow(0)))
	sched.Schedule(scheduler.NewTask(now.Add(time.Second), printNow(1)))
	sched.Schedule(scheduler.NewTask(now.Add(time.Second+500*time.Millisecond), printNow(2)))
	sched.Schedule(scheduler.NewTask(now.Add(2*time.Second+500*time.Millisecond), printNow(3)))
	sched.Schedule(scheduler.NewTask(now.Add(3*time.Second+time.Millisecond), printNow(4)))
	sched.Schedule(scheduler.NewTask(now.Add(4*time.Second+500*time.Millisecond), printNow(5)))
	t, _ := sched.Schedule(scheduler.NewTask(now.Add(4*time.Second+600*time.Millisecond), printNow(6)))
	go func() {
		time.Sleep(4*time.Second + 550*time.Millisecond)
		t.Cancel()
	}()
	sched.Schedule(scheduler.NewTask(now.Add(5*time.Second), printNowWithDelay(7, time.Second)))
	sched.Schedule(scheduler.NewTask(now.Add(5*time.Second+time.Nanosecond), printNowWithDelay(8, time.Second)))
	sched.Schedule(scheduler.NewTask(now.Add(5*time.Second+2*time.Nanosecond), printNowWithDelay(9, time.Second)))
	sched.Schedule(scheduler.NewTask(now.Add(5*time.Second+3*time.Nanosecond), printNowWithDelay(10, time.Second)))
	sched.Schedule(scheduler.NewTask(now.Add(5*time.Second+4*time.Nanosecond), printNowWithDelay(11, time.Second)))
	// These 2 task are delayed because the scheduler limits concurrently processed tasks to 5.
	sched.Schedule(scheduler.NewTask(now.Add(5*time.Second+5*time.Nanosecond), printNowWithDelay(12, time.Second)))
	sched.Schedule(scheduler.NewTask(now.Add(5*time.Second+6*time.Nanosecond), printNowWithDelay(13, time.Second)))

	sched.Schedule(scheduler.NewTask(now.Add(8*time.Second), printNowWithDelay(14, time.Second)))

	ctx, cancel := context.WithDeadline(context.Background(), now.Add(8*time.Second))
	defer func() {
		fmt.Println("calling cancel")
		cancel()
	}()
	err := sched.Start(ctx)
	fmt.Println("Start returned")
	sched.End()
	return err
}
