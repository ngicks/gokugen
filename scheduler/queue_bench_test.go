package scheduler_test

import (
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

func BenchmarkQueue_RemoveCancelled_1000_allCancelled(b *testing.B) {
	b.StopTimer()
	q := scheduler.NewSyncQueue(0)
	now := time.Now()
	for i := 0; i < 1000; i++ {
		t := scheduler.NewTask(now, func(scheduled, current time.Time) {})
		t.Cancel()
		q.Push(t)
	}
	q.Push(scheduler.NewTask(now.Add(-1*time.Second), func(scheduled, current time.Time) {}))

	b.StartTimer()
	q.Exclude(func(ent *scheduler.Task) bool {
		return ent.IsCancelled()
	}, 0, 1000+1)
}
func BenchmarkQueue_RemoveCancelled_10000_halfCancelled(b *testing.B) {
	b.StopTimer()
	q := scheduler.NewSyncQueue(0)
	now := time.Now()
	for i := 0; i < 10000; i++ {
		t := scheduler.NewTask(now, func(scheduled, current time.Time) {})
		if i < 10000/2 {
			t.Cancel()
		}
		q.Push(t)
	}
	q.Push(scheduler.NewTask(now.Add(-1*time.Second), func(scheduled, current time.Time) {}))

	b.StartTimer()
	q.Exclude(func(ent *scheduler.Task) bool {
		return ent.IsCancelled()
	}, 0, 10000+1)
}

func BenchmarkQueue_RemoveCancelled_100000_halfCancelled(b *testing.B) {
	b.StopTimer()
	q := scheduler.NewSyncQueue(0)
	now := time.Now()
	for i := 0; i < 100000; i++ {
		t := scheduler.NewTask(now, func(scheduled, current time.Time) {})
		if i < 100000/2 {
			t.Cancel()
		}
		q.Push(t)
	}
	q.Push(scheduler.NewTask(now.Add(-1*time.Second), func(scheduled, current time.Time) {}))

	b.StartTimer()
	q.Exclude(func(ent *scheduler.Task) bool {
		return ent.IsCancelled()
	}, 0, 100000+1)
}

func BenchmarkQueue_RemoveCancelled_100000_oneCancelled(b *testing.B) {
	b.StopTimer()
	q := scheduler.NewSyncQueue(0)
	now := time.Now()
	for i := 0; i < 100000; i++ {
		t := scheduler.NewTask(now, func(scheduled, current time.Time) {})
		q.Push(t)
	}
	t := scheduler.NewTask(now.Add(-1*time.Second), func(scheduled, current time.Time) {})
	t.Cancel()
	q.Push(t)

	b.StartTimer()
	q.Exclude(func(ent *scheduler.Task) bool {
		return ent.IsCancelled()
	}, 0, 100000+1)
}
