package repository

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/def"
	"github.com/stretchr/testify/require"
)

func TestRepository_tasks_can_be_fetched_by_id(t *testing.T, repo def.Repository) {
	t.Helper()
	require := require.New(t)

	eachState := createEachState(t, repo)

	for _, task := range []def.Task{
		eachState.Scheduled,
		eachState.Cancelled,
		eachState.Dispatched,
		eachState.Done,
		eachState.Err,
	} {
		fetched, err := repo.GetById(context.Background(), task.Id)
		require.NoError(err)
		require.True(
			fetched.Equal(task),
			"not equal. diff = %s", cmp.Diff(task, fetched),
		)
	}

	_, err := repo.GetById(context.Background(), def.NeverExistentId)
	assertIsIdNotFound(t, err)
}
