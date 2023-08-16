package acceptance

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/require"
)

func TestRepository_meta_can_be_updated(t *testing.T, repo def.Repository) {
	t.Helper()
	require := require.New(t)

	tasks := prepareTasks(t, repo)

	ensureChange := func(changed []int) []def.Task {
		return ensureChange(t, repo, tasks, changed)
	}

	type testSet struct {
		query       def.TaskQueryParam
		updateParam map[string]string
		changed     []int // index of tasks which should be updated.
	}

	for _, set := range []testSet{
		{
			query: def.TaskQueryParam{},
		},
	} {
		rowsAffected, err := repo.UpdateMeta(context.Background(), set.query, set.updateParam)
		require.NoError(err)
		require.Equal(len(set.changed), rowsAffected)
		tasks = ensureChange(set.changed)
	}
}

func prepareTasks(t *testing.T, repo def.Repository) []def.Task {
	t.Helper()

	require := require.New(t)

	tasks := make([]def.Task, 0)

	eachState := createEachState(t, repo)

	tasks = append(
		tasks,
		eachState.Scheduled,
		eachState.Cancelled,
		eachState.Dispatched,
		eachState.Done,
		eachState.Err,
	)

	for _, param := range []def.TaskUpdateParam{
		{
			WorkId:      option.Some("baz"),
			Param:       option.Some(map[string]string{"foofoo": "barbar"}),
			Priority:    option.Some(12),
			ScheduledAt: option.Some(sampleDate.Add(3 * time.Hour)),
			Meta:        option.Some(map[string]string{"bazbaz": "corgecorge"}),
		},
		updateParam.Update(def.TaskUpdateParam{
			WorkId: option.Some("baz"),
		}),
		updateParam.Update(def.TaskUpdateParam{
			Param: option.Some(map[string]string{"foofoo": "barbar"}),
		}),
		updateParam.Update(def.TaskUpdateParam{
			Priority: option.Some(12),
		}),
		updateParam.Update(def.TaskUpdateParam{
			ScheduledAt: option.Some(sampleDate.Add(3 * time.Hour)),
		}),
		updateParam.Update(def.TaskUpdateParam{
			Meta: option.Some(map[string]string{"bazbaz": "corgecorge"}),
		}),
		updateParam.Update(def.TaskUpdateParam{
			Meta: option.Some(map[string]string{
				"foofoo": "barbar",
				"bazbaz": "corgecorge",
				"nah":    "boo",
			}),
			Param: option.Some(map[string]string{
				"foofoo": "barbar",
				"bazbaz": "corgecorge",
				"nah":    "boo",
			}),
		}),
		updateParam.Update(def.TaskUpdateParam{
			Meta: option.Some(map[string]string{
				"foofoo": "bar",
				"bazbaz": "corge",
			}),
			Param: option.Some(map[string]string{
				"foofoo": "bar",
				"bazbaz": "corge",
			}),
		}),
		updateParam.Update(def.TaskUpdateParam{
			Meta: option.Some(map[string]string{
				"foofoo": "not_bar",
			}),
			Param: option.Some(map[string]string{
				"foofoo": "not_bar",
			}),
		}),
	} {
		task, err := repo.AddTask(context.Background(), param)
		require.NoError(err)
		tasks = append(tasks, task)
	}

	return tasks
}

// ensureChange ensures tasks are changed or unchaged as
// `changed` is int slice whose contents are indecies of changed tasks of `tasks`.
// As a side effect it re-fetch all tasks and returns them.
func ensureChange(t *testing.T, repo def.Repository, tasks []def.Task, changed []int) []def.Task {
	require := require.New(t)

	refetchedTasks := make([]def.Task, len(tasks))

	for idx, task := range tasks {
		refetched, err := repo.GetById(context.Background(), task.Id)
		require.NoError(err)

		refetchedTasks[idx] = refetched

		if slices.Contains(changed, idx) {
			require.False(task.Equal(refetched))
		} else {
			require.True(task.Equal(refetched))
		}
	}

	return refetchedTasks
}
