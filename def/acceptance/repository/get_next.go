package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/require"
)

func TestRepository_next_task_can_be_fetched(t *testing.T, repo def.Repository) {
	t.Helper()
	require := require.New(t)

	_, err := repo.GetNext(context.Background())
	assertIsExhausted(t, err)

	eachState := createEachState(t, repo)

	task, err := repo.GetNext(context.Background())
	require.NoError(err)
	assertSameTask(t, task, eachState.Scheduled)

	_ = repo.Cancel(context.Background(), task.Id)

	_, err = repo.GetNext(context.Background())
	assertIsExhausted(t, err)

	params := []def.TaskUpdateParam{
		{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(sampleDate),
			Priority:    option.Some(25),
		},
		{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(sampleDate),
			Priority:    option.Some(25),
		},
		{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(sampleDate),
			Priority:    option.Some(20),
		},
		{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(sampleDate),
			Priority:    option.Some(0),
		},
		{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(sampleDate),
			Priority:    option.Some(-2),
		},
		{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(sampleDate.Add(5 * time.Millisecond)),
			Priority:    option.Some(-100),
		},
		{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(sampleDate.Add(time.Second)),
			Priority:    option.Some(0),
		},
		{
			WorkId:      option.Some("foo"),
			ScheduledAt: option.Some(sampleDate.Add(time.Second)),
			Priority:    option.Some(-1),
		},
	}

	// created_at is not related.
	firstTask, err := repo.AddTask(context.Background(), params[0])
	require.NoError(err)
	tasks := createInRandomOrder(t, repo, params[1:])
	tasks = append([]def.Task{firstTask}, tasks...)

	for idx, task := range tasks {
		fetched, err := repo.GetNext(context.Background())
		require.NoError(err)
		if !assertSameTask(t, task, fetched) {
			t.Logf("failed at idx = %d", idx)
		}
		_ = repo.Cancel(context.Background(), fetched.Id)
	}
}

func assertSameTask(t *testing.T, l, r def.Task) bool {
	t.Helper()
	if !l.Equal(r) {
		t.Errorf("not equal. diff = %s", cmp.Diff(l, r))
		return false
	}
	return true
}

func createInRandomOrder(
	t *testing.T,
	repo def.Repository,
	params []def.TaskUpdateParam,
) []def.Task {
	t.Helper()
	require := require.New(t)

	randomIndex := map[int]struct{}{}
	for idx := range params {
		randomIndex[idx] = struct{}{}
	}

	created := make([]def.Task, len(params))

	for idx := range randomIndex {
		task, err := repo.AddTask(context.Background(), params[idx])
		require.NoError(err)
		created[idx] = task
	}

	return created
}
