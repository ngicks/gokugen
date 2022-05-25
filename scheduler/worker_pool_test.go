package scheduler_test

import (
	"testing"

	"github.com/ngicks/gokugen/scheduler"
)

func dummyWorkerFactory() *scheduler.Worker {
	w, _ := scheduler.NewWorker(make(<-chan *scheduler.Task), func() {}, func() {})
	return w
}

func TestWorkerPool(t *testing.T) {
	checker := func(t *testing.T, p *scheduler.WorkerPool) func(totalExpected, wakedExpected, sleepingExpected int) {
		return func(totalExpected, wakedExpected, sleepingExpected int) {
			if total, waked, sleeping := p.Len(); total != totalExpected || waked != wakedExpected || sleeping != sleepingExpected {
				t.Fatalf("worker num is unintended num: "+
					"total[must be %d, is %d], wake[must be %d, is %d], sleeping[must be %d, is %d]",
					totalExpected, total,
					wakedExpected, waked,
					sleepingExpected, sleeping,
				)
			}
		}
	}
	t.Run("basic usage", func(t *testing.T) {
		p := scheduler.NewWorkerPool(dummyWorkerFactory)
		check := checker(t, p)
		check(0, 0, 0)
		p.Add(10)
		check(10, 0, 10)
		p.Wake(3)
		check(10, 3, 7)
		p.Remove(8, false)
		check(2, 2, 0)
		p.Add(10)
		check(12, 2, 10)
		p.Remove(4, true)
		check(8, 0, 8)
		p.Wake(3)
		check(8, 3, 5)
		p.Sleep(1)
		check(8, 2, 6)
		p.Remove(1, true)
		check(7, 1, 6)
		p.Remove(2, false)
		check(5, 1, 4)

		p.Remove(100, false)
		check(0, 0, 0)
		p.Wait()
	})
}
