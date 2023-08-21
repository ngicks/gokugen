package dispatcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ngicks/genericsync"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gommon/pkg/timing"
	"github.com/stretchr/testify/assert"
)

const (
	// if dispatched task is set this work_id, it should block until unblockOneTask is called.
	WorkIdBlocking    = "%%%%%%%%%%%%%%%%%%%%"
	WorkIdError       = "++++++++++++++++++++" //
	WorkIdNoop        = "--------------------"
	WorkIdNonExistent = "~~~~~~~~~~~~~~~~~~~~"
)

var (
	ErrDispatcherSample = errors.New("dispatcher sample")
)

func SetupWorkRegistry() (workRegistry def.WorkRegistry, unblockOneTask func() error) {
	registry := genericsync.Map[string, *def.WorkFn]{}

	ctxChan := make(chan context.Context)
	unblock := func() error {
		ctx := <-ctxChan
		return ctx.Err()
	}

	blockingFn := func(ctx context.Context, param map[string]string) error {
		ctxChan <- ctx
		return nil
	}
	registry.Store(WorkIdBlocking, &blockingFn)

	noopFn := func(ctx context.Context, param map[string]string) error {
		return nil
	}
	registry.Store(WorkIdNoop, &noopFn)

	errFn := func(ctx context.Context, param map[string]string) error {
		return ErrDispatcherSample
	}
	registry.Store(WorkIdError, &errFn)

	return &registry, unblock
}

type DispatcherTestOption struct {
	Debug          bool
	Max            int // max concurrent task of given dispatcher impl.
	UnblockOneTask func() error
}

func TestDispatcher(t *testing.T, dispatcher def.Dispatcher, opt DispatcherTestOption) {
	t.Run("block at max concurrent plus 1 tasks", func(t *testing.T) {
		testDispatcher_block_at_max_concurrent_plus_1_tasks(t, dispatcher, opt)
	})

	t.Run("returns context.Err() if it is already cancelled", func(t *testing.T) {
		testDispatcher_returns_context_Err_if_it_is_already_cancelled(t, dispatcher, opt)
	})

	t.Run("propagates cancellation", func(t *testing.T) {
		testDispatcher_propagates_cancellation(t, dispatcher, opt)
	})

	t.Run("returns ErrWorkIdNotFound if unknown work_id", func(t *testing.T) {
		testDispatcher_returns_ErrWorkIdNotFound_if_unknown_work_id(t, dispatcher, opt)
	})

}

func testDispatcher_block_at_max_concurrent_plus_1_tasks(
	t *testing.T,
	dispatcher def.Dispatcher,
	opt DispatcherTestOption,
) {
	assert := assert.New(t)

	retChannels := make([](<-chan error), opt.Max)
	for i := 0; i < opt.Max; i++ {
		errChan, err := dispatcher.Dispatch(
			context.Background(),
			func(ctx context.Context) (def.Task, error) {
				return def.Task{
					WorkId: WorkIdBlocking,
				}, nil
			},
		)

		assert.NoError(err)
		retChannels[i] = errChan
	}

	waiter := timing.CreateWaiterCh(func() {
		errCh, err := dispatcher.Dispatch(
			context.Background(),
			func(ctx context.Context) (def.Task, error) {
				return def.Task{
					WorkId: WorkIdBlocking,
				}, nil
			},
		)
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

	_ = opt.UnblockOneTask()

	// at least one.
	recvOne := func() {
	tryRecvLoop:
		for {
			for _, ch := range retChannels {
				select {
				case <-ch:
					break tryRecvLoop
				default:
				}
			}
		}
	}

	recvOne()

	defer func() {
		for i := 0; i < opt.Max-1; i++ {
			recvOne()
		}
	}()

	select {
	case <-waiter:
	case <-time.After(time.Millisecond):
		t.Errorf("Dispatch must not block at this point")
	}

	for i := 0; i < opt.Max; i++ {
		_ = opt.UnblockOneTask()
	}
}

func testDispatcher_returns_context_Err_if_it_is_already_cancelled(
	t *testing.T,
	dispatcher def.Dispatcher,
	opt DispatcherTestOption,
) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	errCh, err := dispatcher.Dispatch(ctx, func(ctx context.Context) (def.Task, error) {
		return def.Task{
			WorkId: WorkIdNoop,
		}, nil
	})

	assert.Nil(errCh)
	assert.ErrorIs(err, context.Canceled)
}
func testDispatcher_propagates_cancellation(
	t *testing.T,
	dispatcher def.Dispatcher,
	opt DispatcherTestOption,
) {
	assert := assert.New(t)

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
			errCh, err = dispatcher.Dispatch(ctx, func(ctx context.Context) (def.Task, error) {
				<-stepChan
				<-stepChan
				<-ctx.Done()
				assert.Error(ctx.Err(), "iter %d", i)
				return def.Task{
					WorkId: WorkIdNoop,
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

	ctx, cancel := context.WithCancel(context.Background())
	errCh, err := dispatcher.Dispatch(ctx, func(ctx context.Context) (def.Task, error) {
		return def.Task{
			WorkId: WorkIdBlocking,
		}, nil
	})
	assert.NoError(err)
	cancel()
	fnCtxErr := opt.UnblockOneTask()
	assert.Error(fnCtxErr)
	assert.NoError(<-errCh)
}

func testDispatcher_returns_ErrWorkIdNotFound_if_unknown_work_id(
	t *testing.T,
	dispatcher def.Dispatcher,
	opt DispatcherTestOption,
) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh, err := dispatcher.Dispatch(ctx, func(ctx context.Context) (def.Task, error) {
		return def.Task{
			WorkId: WorkIdNonExistent,
		}, nil
	})

	assert.NotNil(errCh)
	assert.NoError(err)
	workErr := <-errCh
	assert.Error(workErr)
	assert.ErrorIs(workErr, def.ErrWorkIdNotFound)
}
