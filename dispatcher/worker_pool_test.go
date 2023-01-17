package dispatcher

import (
	"context"
	"testing"

	"github.com/ngicks/gokugen/scheduler"
	acceptancetest "github.com/ngicks/gokugen/scheduler/acceptance_test"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

func setUpWorkerPool() (dispatcher *WorkerPoolDispatcher, unblockOneTask func()) {
	workRegistry := syncparam.Map[string, scheduler.WorkFn]{}

	stepChan := make(chan struct{})
	unblock := func() {
		stepChan <- struct{}{}
	}
	workRegistry.Store(acceptancetest.DispatcherBlockingWorker, func(ctx context.Context, param []byte) error {
		<-stepChan
		return nil
	})
	workRegistry.Store(acceptancetest.DispatcherNoopWork, func(ctx context.Context, param []byte) error {
		return nil
	})
	workRegistry.Store(acceptancetest.DispatcherErrorWorker, func(ctx context.Context, param []byte) error {
		return acceptancetest.ErrDispatcherSample
	})

	workerPoolDispatcher := NewWorkerPoolDispatcher(&workRegistry)

	return workerPoolDispatcher, unblock
}
func TestWorkerPoolDispatcherAcceptance_limited(t *testing.T) {
	workerPoolDispatcher, unblockOneTask := setUpWorkerPool()

	workerPoolDispatcher.WorkerPool.Add(5)
	workerPoolDispatcher.workerPool.WaitUntil(func(alive, sleeping, active int) bool {
		return alive == 5
	})

	defer workerPoolDispatcher.WorkerPool.Wait()
	defer workerPoolDispatcher.WorkerPool.Remove(100)

	acceptancetest.TestLimitedDispatcher(t, workerPoolDispatcher, unblockOneTask)
}

func TestWorkerPoolDispatcherAcceptance(t *testing.T) {
	workerPoolDispatcher, unblockOneTask := setUpWorkerPool()

	workerPoolDispatcher.WorkerPool.Add(10)
	workerPoolDispatcher.workerPool.WaitUntil(func(alive, sleeping, active int) bool {
		return alive == 10
	})

	defer workerPoolDispatcher.WorkerPool.Wait()
	defer workerPoolDispatcher.WorkerPool.Remove(150)

	acceptancetest.TestDispatcher(t, workerPoolDispatcher, unblockOneTask)
}
