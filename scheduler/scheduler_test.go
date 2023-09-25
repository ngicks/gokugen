package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ngicks/gokugen/repository/inmemory"
	"github.com/ngicks/mockable"
	"github.com/stretchr/testify/require"
)

var (
	mockErr = fmt.Errorf("foo")
)

func TestScheduler(t *testing.T) {
	require := require.New(t)

	mem := inmemory.NewInMemoryRepository()
	repo := &mockRepository{
		InMemoryRepository: mem,
		Clock:              *mockable.NewClockFake(time.Now()),
	}

	dispatcher := &mockDispatcher{
		Pending: make(map[string]pair),
	}

	scheduler := NewScheduler(repo, dispatcher)

	repo.TimerErr = mockErr
	state := scheduler.Step(context.Background())

	require.Equal(TimerUpdateError, state.State())
}
