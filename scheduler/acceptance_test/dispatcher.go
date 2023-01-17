package acceptancetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/gommon/pkg/timing"
	"github.com/stretchr/testify/assert"
)

const (
	DispatcherBlockingWorker = "%%%%%%%%%%%%%%%%%%%%"
	DispatcherErrorWorker    = "++++++++++++++++++++"
	DispatcherNoopWork       = "--------------------"
)

var (
	ErrDispatcherSample = errors.New("dispatcher sample")
)

// TestLimitedDispatcher tests a limited dispatcher interface.
// Implementations may skip this test if it does not limits it number of concurrently worked tasks.
//
// Dispatcher is expected to have 5 workers.
func TestLimitedDispatcher(t *testing.T, dispatcher scheduler.Dispatcher, unblockOneTask func()) {
	t.Run("Dispatcher blocks at 6th task", func(t *testing.T) {
		assert := assert.New(t)

		var retChannels [5](<-chan error)
		for i := 0; i < 5; i++ {
			errChan, err := dispatcher.Dispatch(context.Background(), func(ctx context.Context) (scheduler.Task, error) {
				return scheduler.Task{
					WorkId: DispatcherBlockingWorker,
				}, nil
			})

			assert.NoError(err)
			retChannels[i] = errChan
		}

		waiter := timing.CreateWaiterCh(func() {
			errCh, err := dispatcher.Dispatch(context.Background(), func(ctx context.Context) (scheduler.Task, error) {
				return scheduler.Task{
					WorkId: DispatcherBlockingWorker,
				}, nil
			})
			assert.NoError(err)
			go func() {
				<-errCh
			}()
		})

		select {
		case <-waiter:
			t.Errorf("must block on Dispatch")
		case <-time.After(time.Millisecond):
		}

		unblockOneTask()

		// at least one.
		recvOne := func() {
			select {
			case <-retChannels[0]:
			case <-retChannels[1]:
			case <-retChannels[2]:
			case <-retChannels[3]:
			case <-retChannels[4]:
			}
		}

		recvOne()

		defer func() {
			for i := 0; i < 4; i++ {
				recvOne()
			}
		}()

		select {
		case <-waiter:
		case <-time.After(time.Millisecond):
			t.Errorf("Dispatch must not block at this point")
		}

		for i := 0; i < 5; i++ {
			unblockOneTask()
		}
	})
}

// TestDispatcher tests a dispatcher interface.
// Implementations call this test from within their package code.
func TestDispatcher(t *testing.T, dispatcher scheduler.Dispatcher, unblockOneTask func()) {
	assert := assert.New(t)

	t.Run("already cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		errCh, err := dispatcher.Dispatch(ctx, func(ctx context.Context) (scheduler.Task, error) {
			return scheduler.Task{
				WorkId: DispatcherNoopWork,
			}, nil
		})

		assert.Nil(errCh)
		assert.ErrorIs(err, context.Canceled)
	})

	t.Run("cancel propagation", func(t *testing.T) {
		for i, withFn := range [](func(context.Context) (context.Context, context.CancelFunc)){
			context.WithCancel,
			func(ctx context.Context) (context.Context, context.CancelFunc) {
				c, cancel := context.WithTimeout(ctx, 2*time.Millisecond)
				return c, func() { <-c.Done(); cancel() }
			},
		} {
			ctx, cancel := withFn(context.Background())
			defer cancel()
			stepChan := make(chan struct{})

			var errCh <-chan error
			var err error
			waiter := timing.CreateWaiterFn(func() {
				errCh, err = dispatcher.Dispatch(ctx, func(ctx context.Context) (scheduler.Task, error) {
					<-stepChan
					<-stepChan
					<-ctx.Done()
					assert.Error(ctx.Err(), "iter %d", i)
					return scheduler.Task{
						WorkId: DispatcherNoopWork,
					}, nil
				})
			})

			stepChan <- struct{}{}
			cancel()
			stepChan <- struct{}{}
			waiter()
			assert.NoError(err, "iter %d", i)
			assert.NotNil(errCh, "iter %d", i)
			assert.Error(<-errCh, "iter %d", i)
		}
	})

}
