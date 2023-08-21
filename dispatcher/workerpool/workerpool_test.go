package workerpool

import (
	"testing"

	"github.com/ngicks/gokugen/def/acceptance/dispatcher"
)

func TestWorkerPool(t *testing.T) {
	reg, unblockOneTask := dispatcher.SetupWorkRegistry()

	workerPoolDispatcher := NewWorkerPoolDispatcher(reg)

	workerPoolDispatcher.WorkerPool.Add(10)
	workerPoolDispatcher.WorkerPool.WaitUntil(func(alive, sleeping, active int) bool {
		return alive == 10
	})

	defer workerPoolDispatcher.WorkerPool.Wait()
	defer workerPoolDispatcher.WorkerPool.Remove(100)

	dispatcher.TestDispatcher(t, workerPoolDispatcher, dispatcher.DispatcherTestOption{
		Debug:          true,
		Max:            10,
		UnblockOneTask: unblockOneTask,
	})

	workerPoolDispatcher.workerPool.Remove(5)
	workerPoolDispatcher.WorkerPool.WaitUntil(func(alive, sleeping, active int) bool {
		return alive == 5
	})

	dispatcher.TestDispatcher(t, workerPoolDispatcher, dispatcher.DispatcherTestOption{
		Debug:          true,
		Max:            5,
		UnblockOneTask: unblockOneTask,
	})
}
