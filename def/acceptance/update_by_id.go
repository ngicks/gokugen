package acceptance

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
	testRepository_state_other_than_scheduled_is_not_updatable(t, repo)
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
	require.True(task.CancelledAt.Equal(updated.CancelledAt))
	require.True(task.DispatchedAt.Equal(updated.DispatchedAt))
	require.True(task.DoneAt.Equal(updated.DoneAt))
	require.True(maps.Equal(updateParam.Param.Value(), updated.Param))
	require.True(maps.Equal(updateParam.Meta.Value(), updated.Meta))

	require.True(
		task.Update(updateParamPartial, true).Equal(updated),
		"not equal. diff = %s", cmp.Diff(task.Update(updateParamPartial, true), updated),
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
			task.Update(updateParam, true).Equal(updated),
			"not equal. diff = %s", cmp.Diff(task.Update(updateParam, true), updated),
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

func testRepository_state_other_than_scheduled_is_not_updatable(t *testing.T, repo def.Repository) {
	t.Helper()
	require := require.New(t)

	eachState := createEachState(t, repo)

	var err error
	err = repo.UpdateById(context.Background(), eachState.Cancelled.Id, updateParam)
	require.True(
		def.IsAlreadyCancelled(err),
		"incorrect error. must be def.IsAlreadyCancelled(err) == true, but is %s", err,
	)

	err = repo.UpdateById(context.Background(), eachState.Dispatched.Id, updateParam)
	require.True(
		def.IsAlreadyDispatched(err),
		"incorrect error. must be def.IsAlreadyDispatched(err) == true, but is %s", err,
	)

	err = repo.UpdateById(context.Background(), eachState.Done.Id, updateParam)
	require.True(
		def.IsAlreadyDone(err),
		"incorrect error. must be def.IsAlreadyDone(err) == true, but is %s", err,
	)

	err = repo.UpdateById(context.Background(), eachState.Err.Id, updateParam)
	require.True(
		def.IsAlreadyDone(err),
		"incorrect error. must be def.IsAlreadyDone(err) == true, but is %s", err,
	)
}
