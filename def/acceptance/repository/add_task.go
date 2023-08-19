package repository

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/def/util"
	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/require"
)

// It ensures
//
//   - All possible None / Some combination is acceptable
//   - Id's are auto-generated and not overlapping at least for 300 elements
//   - An user can fetch added tasks by calling GetById.
//   - It returns ErrInvalidTask if some specific field of TaskUpdateParam is empty.
func TestRepository_tasks_can_be_added(t *testing.T, repo def.Repository) {
	t.Helper()

	testRepository_add_task_1500_elements(t, repo)
	testRepository_add_task_error(t, repo)
	testRepository_add_task_with_nil_replaced_with_empty_map(t, repo)
	testRepository_add_task_normalize(t, repo)
}

func testRepository_add_task_1500_elements(t *testing.T, repo def.Repository) {
	t.Helper()

	require := require.New(t)

	ctx := context.Background()

	idSet := make(map[string]struct{})

	for i := 0; i < 100; i++ {
		for i := 1; i <= 0b1111; i++ {
			// shadowing
			taskParam := initialParam.Clone()

			if i&0b0001 == 0 {
				taskParam.Param = option.None[map[string]string]()
			}
			if i&0b0010 == 0 {
				taskParam.Priority = option.None[int]()
			}
			if i&0b0100 == 0 {
				taskParam.Deadline = option.None[option.Option[time.Time]]()
			}
			if i&0b1000 == 0 {
				taskParam.Meta = option.None[map[string]string]()
			}

			task := testRepository_add_task(t, repo, taskParam)

			if _, alreadyExist := idSet[task.Id]; alreadyExist {
				t.Fatalf("AddTask returned same id twice. task = %+#v", task)
			}
			idSet[task.Id] = struct{}{}

			fetched, err := repo.GetById(ctx, task.Id)
			require.NoError(err)
			require.True(fetched.Equal(task), "not equal. diff = %s", cmp.Diff(fetched, task))
		}
	}
}

func testRepository_add_task(
	t *testing.T,
	repo def.Repository,
	initialParam def.TaskUpdateParam,
) def.Task {
	t.Helper()

	require := require.New(t)

	ctx := context.Background()

	timeBeforeAddition := util.DropMicros(time.Now())

	task, err := repo.AddTask(ctx, initialParam)
	require.NoError(err)

	require.NotEmpty(task.Id)
	require.Equal(initialParam.WorkId.Value(), task.WorkId)
	require.Equal(initialParam.Priority.Value(), task.Priority)
	require.Equal(def.TaskScheduled, task.State)
	// param, meta
	require.True(maps.Equal(initialParam.Param.Value(), task.Param))
	require.True(maps.Equal(initialParam.Meta.Value(), task.Meta))

	// time fields
	require.True(sampleDate.Equal(task.ScheduledAt))
	// Theoretically task.CreatedAt.Compare(currentTime) == 1 is always true.
	// However some test environment does not allow the timer to advance.
	// It is a wrong assumption for those environments.
	require.GreaterOrEqual(task.CreatedAt.Compare(timeBeforeAddition), 0)
	require.True(initialParam.Deadline.Value().Equal(task.Deadline))
	require.True(task.CancelledAt.IsNone())
	require.True(task.DispatchedAt.IsNone())
	require.True(task.DoneAt.IsNone())
	require.Empty(task.Err)

	return task
}

func testRepository_add_task_error(t *testing.T, repo def.Repository) {
	t.Helper()

	require := require.New(t)

	for _, param := range []def.TaskUpdateParam{
		{},
		{
			WorkId: option.Some(""),
		},
		{
			ScheduledAt: option.Some(time.Time{}),
		},
		{
			Param:       option.Some(map[string]string{"bar": "baz"}),
			Priority:    option.Some(12),
			ScheduledAt: option.Some(sampleDate),
			Meta:        option.Some(map[string]string{"qux": "quux"}),
		},
		{
			WorkId: option.Some("foo"),
		},
	} {
		_, err := repo.AddTask(context.Background(), param)
		require.ErrorIs(err, def.ErrInvalidTask)
	}
}

func testRepository_add_task_with_nil_replaced_with_empty_map(t *testing.T, repo def.Repository) {
	t.Helper()

	require := require.New(t)

	task, err := repo.AddTask(context.Background(), def.TaskUpdateParam{
		WorkId:      option.Some("nah"),
		ScheduledAt: option.Some(sampleDate),
		Param:       option.Some[map[string]string](nil),
		Meta:        option.Some[map[string]string](nil),
	})
	require.NoError(err)

	require.True(maps.Equal(task.Param, map[string]string{}))
	require.NotNil(task.Param)
	require.True(maps.Equal(task.Param, map[string]string{}))
	require.NotNil(task.Meta)
}

func testRepository_add_task_normalize(t *testing.T, repo def.Repository) {
	t.Helper()
	require := require.New(t)

	task, err := repo.AddTask(
		context.Background(),
		initialParam.Update(def.TaskUpdateParam{
			ScheduledAt: option.Some(exampleDateNotNormalized),
			Deadline:    option.Some(option.Some(exampleDateNotNormalized.Add(3 * time.Minute))),
		}),
	)
	require.NoError(err)

	require.True(task.ScheduledAt.Equal(exampleDateNormalized))
	require.True(task.Deadline.Value().Equal(exampleDateNormalized.Add(3 * time.Minute)))
}
