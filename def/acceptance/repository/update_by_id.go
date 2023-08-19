package repository

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/require"
)

func TestRepository_tasks_can_be_updated(t *testing.T, repo def.Repository) {
	t.Helper()

	// first of all, update with known value and
	// check if task.Update matches behavior of UpdateById.
	testRepository_update_to_known_value(t, repo)
	testRepository_update_with_all_possible_none_some_combination(t, repo)
	testRepository_update_with_nil_replaced_with_empty_map(t, repo)
	testRepository_update_nonexistent(t, repo)
	testRepository_update_state_other_than_scheduled_is_not_updatable(t, repo)
	testRepository_update_normalize(t, repo)
}

func testRepository_update_to_known_value(t *testing.T, repo def.Repository) {
	t.Helper()

	require := require.New(t)

	task, err := repo.AddTask(context.Background(), initialParam)
	require.NoError(err)

	updateParamPartial := updateParam
	updateParamPartial.Priority = option.None[int]()
	err = repo.UpdateById(context.Background(), task.Id, updateParamPartial)
	require.NoError(err)

	updated, err := repo.GetById(
		context.Background(),
		task.Id,
	)
	require.NoError(err)

	require.Equal(task.Id, updated.Id)
	require.Equal(updateParam.WorkId.Value(), updated.WorkId)
	require.Equal(initialParam.Priority.Value(), updated.Priority)
	require.Equal(def.TaskScheduled, updated.State)
	require.Empty(updated.Err)
	require.True(updateParam.ScheduledAt.Value().Equal(updated.ScheduledAt))
	require.True(task.CreatedAt.Equal(updated.CreatedAt))
	require.True(updateParam.Deadline.Value().Equal(updated.Deadline))
	require.True(task.CancelledAt.Equal(updated.CancelledAt))
	require.True(task.DispatchedAt.Equal(updated.DispatchedAt))
	require.True(task.DoneAt.Equal(updated.DoneAt))
	require.True(maps.Equal(updateParam.Param.Value(), updated.Param))
	require.True(maps.Equal(updateParam.Meta.Value(), updated.Meta))

	require.True(
		task.Update(updateParamPartial).Equal(updated),
		"not equal. diff = %s", cmp.Diff(task.Update(updateParamPartial), updated),
	)
}

func testRepository_update_with_all_possible_none_some_combination(
	t *testing.T,
	repo def.Repository,
) {
	t.Helper()

	require := require.New(t)

	for i := 1; i <= 0b11111; i++ {
		task, err := repo.AddTask(context.Background(), initialParam)
		require.NoError(err)

		updateParam := updateParam
		if i&0b00001 == 0 {
			updateParam.WorkId = option.None[string]()
		}
		if i&0b00010 == 0 {
			updateParam.Param = option.None[map[string]string]()
		}
		if i&0b00100 == 0 {
			updateParam.Priority = option.None[int]()
		}
		if i&0b01000 == 0 {
			updateParam.ScheduledAt = option.None[time.Time]()
		}
		if i&0b10000 == 0 {
			updateParam.Meta = option.None[map[string]string]()
		}

		err = repo.UpdateById(context.Background(), task.Id, updateParam)
		require.NoError(err)

		updated, err := repo.GetById(context.Background(), task.Id)
		require.NoError(err)
		require.True(
			task.Update(updateParam).Equal(updated),
			"not equal. diff = %s", cmp.Diff(task.Update(updateParam), updated),
		)
	}
}

func testRepository_update_with_nil_replaced_with_empty_map(t *testing.T, repo def.Repository) {
	t.Helper()

	require := require.New(t)

	task, err := repo.AddTask(context.Background(), initialParam)
	require.NoError(err)
	err = repo.UpdateById(
		context.Background(),
		task.Id, def.TaskUpdateParam{
			Param: option.Some[map[string]string](nil),
			Meta:  option.Some[map[string]string](nil),
		},
	)
	require.NoError(err)

	updated, err := repo.GetById(context.Background(), task.Id)
	require.NoError(err)
	require.True(maps.Equal(updated.Param, map[string]string{}))
	require.NotNil(updated.Param)
	require.True(maps.Equal(updated.Param, map[string]string{}))
	require.NotNil(updated.Meta)
}

func testRepository_update_nonexistent(t *testing.T, repo def.Repository) {
	t.Helper()

	err := repo.UpdateById(context.Background(), def.NeverExistentId, updateParam)
	assertIsIdNotFound(t, err)
}

func testRepository_update_state_other_than_scheduled_is_not_updatable(
	t *testing.T,
	repo def.Repository,
) {
	t.Helper()

	eachState := CreateEachState(t, repo)

	var err error
	err = repo.UpdateById(context.Background(), eachState.Cancelled.Id, updateParam)
	assertIsAlreadyCancelled(t, err)

	err = repo.UpdateById(context.Background(), eachState.Dispatched.Id, updateParam)
	assertIsAlreadyDispatched(t, err)

	err = repo.UpdateById(context.Background(), eachState.Done.Id, updateParam)
	assertIsAlreadyDone(t, err)

	err = repo.UpdateById(context.Background(), eachState.Err.Id, updateParam)
	assertIsAlreadyDone(t, err)
}

func testRepository_update_normalize(t *testing.T, repo def.Repository) {
	t.Helper()
	require := require.New(t)

	task, err := repo.AddTask(context.Background(), initialParam)
	require.NoError(err)
	err = repo.UpdateById(context.Background(), task.Id, def.TaskUpdateParam{
		ScheduledAt: option.Some(exampleDateNotNormalized),
		Deadline:    option.Some(option.Some(exampleDateNotNormalized.Add(3 * time.Minute))),
	})
	require.NoError(err)

	task, err = repo.GetById(context.Background(), task.Id)
	require.NoError(err)

	require.True(
		task.ScheduledAt.Equal(exampleDateNormalized),
		"expected = %s, actual = %s", exampleDateNormalized, task.ScheduledAt,
	)
	assertTimeNormalized(t, task.Deadline.Value())
}
