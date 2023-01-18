package scheduler_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ngicks/gokugen/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestIsRepositoryError(t *testing.T) {
	assert := assert.New(t)

	// Test Case struct
	type tc struct {
		fn     func(error) bool
		input  error
		result bool
		label  string
	}

	errSample := errors.New("sample")

	for _, testCase := range []tc{
		{
			scheduler.IsAlreadyCancelled,
			&scheduler.RepositoryError{Kind: scheduler.AlreadyCancelled},
			true,
			"Direct already cancelled",
		},
		{
			scheduler.IsAlreadyDispatched,
			&scheduler.RepositoryError{Kind: scheduler.AlreadyDispatched},
			true,
			"Direct already dispatched",
		},
		{
			scheduler.IsAlreadyDone,
			&scheduler.RepositoryError{Kind: scheduler.AlreadyDone},
			true,
			"Direct already done",
		},
		{
			scheduler.IsEmpty,
			&scheduler.RepositoryError{Kind: scheduler.Empty},
			true,
			"Direct already empty",
		},
		{
			scheduler.IsIdNotFound,
			&scheduler.RepositoryError{Kind: scheduler.IdNotFound},
			true,
			"Direct already id not found",
		},
		{
			scheduler.IsNotDispatched,
			&scheduler.RepositoryError{Kind: scheduler.NotDispatched},
			true,
			"Direct already not dispatched",
		},
	} {
		assert.Equal(testCase.result, testCase.fn(testCase.input), testCase.label)
		assert.False(testCase.fn(errSample))
	}

	assert.True(
		scheduler.IsRepositoryErr(
			&scheduler.RepositoryError{Kind: scheduler.AlreadyCancelled},
			scheduler.AlreadyCancelled,
		),
	)
	assert.False(
		scheduler.IsRepositoryErr(
			&scheduler.RepositoryError{Kind: scheduler.AlreadyCancelled},
			scheduler.AlreadyDispatched,
		),
	)
	assert.False(
		scheduler.IsRepositoryErr(
			errSample,
			scheduler.AlreadyCancelled,
		),
	)
	assert.False(
		scheduler.IsRepositoryErr(
			nil,
			scheduler.AlreadyCancelled,
		),
	)

	repoErr := &scheduler.RepositoryError{Kind: scheduler.AlreadyCancelled}
	wrapped := fmt.Errorf("%w", repoErr)
	nonRepoErrWrapped := fmt.Errorf("%w", errSample)
	for i := 0; i < 10; i++ {
		assert.True(
			scheduler.IsAlreadyCancelled(wrapped),
			"wrapped error must return true if it has RepositoryError in its chain.",
		)
		assert.False(scheduler.IsAlreadyCancelled(nonRepoErrWrapped))
		wrapped = fmt.Errorf("%w", wrapped)
		nonRepoErrWrapped = fmt.Errorf("%w", nonRepoErrWrapped)
	}
}
