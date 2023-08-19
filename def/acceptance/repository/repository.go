package repository

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
)

var (
	jst *time.Location

	sampleDate = time.Date(
		2023,
		time.April,
		23,
		17,
		24,
		46,
		123000000, // it ignores under micro secs.
		time.UTC,
	)

	exampleDateNotNormalized time.Time
	exampleDateNormalized    = time.Date(1992, 8, 17, 2, 15, 24, 123000000, time.UTC)

	initialParam = def.TaskUpdateParam{
		WorkId:      option.Some("foo"),
		Param:       option.Some(map[string]string{"bar": "baz"}),
		Priority:    option.Some(12),
		ScheduledAt: option.Some(sampleDate),
		Deadline:    option.Some(option.Some(sampleDate.Add(5 * time.Minute))),
		Meta:        option.Some(map[string]string{"qux": "quux"}),
	}

	updateParam = def.TaskUpdateParam{
		WorkId:      option.Some("bar"),
		Param:       option.Some(map[string]string{"corge": "grault", "garply": "waldo"}),
		Priority:    option.Some(-5),
		ScheduledAt: option.Some(sampleDate.Add(time.Hour)),
		Deadline:    option.Some(option.Some(sampleDate.Add(10 * time.Minute))),
		Meta:        option.Some(map[string]string{"fred": "plugh"}),
	}
)

func init() {
	jst, _ = time.LoadLocation("Asia/Tokyo")
	exampleDateNotNormalized = time.Date(1992, 8, 17, 11, 15, 24, 123456789, jst)
}

// TestRepository is an acceptance test set for Repository interface.
// Only those which can pass this test are considered conformers of the interface.
//
// This test uses repository instances returned from newInitializedRepository.
// The function will be called multiple times.
// The test uses each instance however only one instance is used at a time.
// Therefore an invocation may invalidate all instances it has returned.
func TestRepository(t *testing.T, newInitializedRepository func() def.Repository) {
	t.Helper()

	t.Run("tasks can be added", func(t *testing.T) {
		TestRepository_tasks_can_be_added(t, newInitializedRepository())
	})

	t.Run("tasks can be fetched by id", func(t *testing.T) {
		TestRepository_tasks_can_be_fetched_by_id(t, newInitializedRepository())
	})

	t.Run("tasks can be updated", func(t *testing.T) {
		TestRepository_tasks_can_be_updated(t, newInitializedRepository())
	})

	t.Run("tasks can be cancelled", func(t *testing.T) {
		TestRepository_tasks_can_be_cancelled(t, newInitializedRepository())
	})

	t.Run("tasks can be marked as dispatched", func(t *testing.T) {
		TestRepository_tasks_can_be_marked_as_dispatched(t, newInitializedRepository())
	})

	t.Run("dispatched tasks can be marked as done", func(t *testing.T) {
		TestRepository_dispatched_tasks_can_be_marked_as_done(t, newInitializedRepository())
	})

	t.Run("can find tasks", func(t *testing.T) {
		TestRepository_can_find_tasks(t, newInitializedRepository())
	})

	t.Run("next task can be fetched", func(t *testing.T) {
		TestRepository_next_task_can_be_fetched(t, newInitializedRepository())
	})
}

func assertTimeNormalized(t *testing.T, v time.Time) bool {
	t.Helper()

	didErr := false
	if v.Nanosecond()%1000000 != 0 {
		didErr = true
		t.Errorf(
			"time must be truncated to milli secs, but has micro or finer value = %d",
			v.Nanosecond(),
		)
	}
	if v.Location() != time.UTC {
		didErr = true
		t.Errorf("location must be time.UTC. but is %s", v.Location())
	}
	return !didErr
}

func assertIsExhausted(t *testing.T, err error) bool {
	t.Helper()

	if !def.IsExhausted(err) {
		t.Errorf(
			"incorrect error. must be def.IsExhausted(err) == true, but is %s", err,
		)
		return false
	}
	return true
}

func assertIsIdNotFound(t *testing.T, err error) bool {
	t.Helper()

	if !def.IsIdNotFound(err) {
		t.Errorf(
			"incorrect error. must be def.IsIdNotFound(err) == true, but is %s", err,
		)
		return false
	}
	return true
}

func assertIsNotDispatched(t *testing.T, err error) bool {
	t.Helper()

	if !def.IsNotDispatched(err) {
		t.Errorf(
			"incorrect error. must be def.IsNotDispatched(err) == true, but is %s", err,
		)
		return false
	}
	return true
}

func assertIsAlreadyCancelled(t *testing.T, err error) bool {
	t.Helper()

	if !def.IsAlreadyCancelled(err) {
		t.Errorf(
			"incorrect error. must be def.IsAlreadyCancelled(err) == true, but is %s", err,
		)
		return false
	}
	return true
}

func assertIsAlreadyDispatched(t *testing.T, err error) bool {
	t.Helper()

	if !def.IsAlreadyDispatched(err) {
		t.Errorf(
			"incorrect error. must be def.IsAlreadyDispatched(err) == true, but is %s", err,
		)
		return false
	}
	return true
}

func assertIsAlreadyDone(t *testing.T, err error) bool {
	t.Helper()

	if !def.IsAlreadyDone(err) {
		t.Errorf(
			"incorrect error. must be def.IsAlreadyDone(err) == true, but is %s", err,
		)
		return false
	}
	return true
}
