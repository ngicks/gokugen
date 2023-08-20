package def

import (
	"errors"
	"fmt"
	"testing"

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
			IsAlreadyCancelled,
			&RepositoryError{Kind: AlreadyCancelled},
			true,
			"Direct already cancelled",
		},
		{
			IsAlreadyDispatched,
			&RepositoryError{Kind: AlreadyDispatched},
			true,
			"Direct already dispatched",
		},
		{
			IsAlreadyDone,
			&RepositoryError{Kind: AlreadyDone},
			true,
			"Direct already done",
		},
		{
			IsExhausted,
			&RepositoryError{Kind: Exhausted},
			true,
			"Direct already empty",
		},
		{
			IsIdNotFound,
			&RepositoryError{Kind: IdNotFound},
			true,
			"Direct already id not found",
		},
		{
			IsNotDispatched,
			&RepositoryError{Kind: NotDispatched},
			true,
			"Direct already not dispatched",
		},
	} {
		assert.Equal(testCase.result, testCase.fn(testCase.input), testCase.label)
		assert.False(testCase.fn(errSample))
	}

	assert.True(
		IsRepositoryErr(
			&RepositoryError{Kind: AlreadyCancelled},
			AlreadyCancelled,
		),
	)
	assert.False(
		IsRepositoryErr(
			&RepositoryError{Kind: AlreadyCancelled},
			AlreadyDispatched,
		),
	)
	assert.False(
		IsRepositoryErr(
			errSample,
			AlreadyCancelled,
		),
	)
	assert.False(
		IsRepositoryErr(
			nil,
			AlreadyCancelled,
		),
	)

	repoErr := &RepositoryError{Kind: AlreadyCancelled}
	wrapped := fmt.Errorf("%w", repoErr)
	nonRepoErrWrapped := fmt.Errorf("%w", errSample)
	for i := 0; i < 30; i++ {
		assert.True(
			IsAlreadyCancelled(wrapped),
			"wrapped error must return true if it has RepositoryError in its chain.",
		)
		assert.False(IsAlreadyCancelled(nonRepoErrWrapped))
		if i%3 != 0 {
			wrapped = fmt.Errorf("%w", wrapped)
		} else {
			wrapped = fmt.Errorf("%w, %w", wrapped, errors.New("mah"))
		}
		nonRepoErrWrapped = fmt.Errorf("%w", nonRepoErrWrapped)
	}
}
