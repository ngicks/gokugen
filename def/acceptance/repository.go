package acceptance

import (
	"testing"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
)

var (
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

	initialParam = def.TaskUpdateParam{
		WorkId:      option.Some("foo"),
		Param:       option.Some(map[string]string{"bar": "baz"}),
		Priority:    option.Some(12),
		ScheduledAt: option.Some(sampleDate),
		Meta:        option.Some(map[string]string{"qux": "quux"}),
	}

	updateParam = def.TaskUpdateParam{
		WorkId:      option.Some("bar"),
		Param:       option.Some(map[string]string{"corge": "grault", "garply": "waldo"}),
		Priority:    option.Some(-5),
		ScheduledAt: option.Some(sampleDate.Add(time.Hour)),
		Meta:        option.Some(map[string]string{"fred": "plugh"}),
	}
)

// TestRepository is an accecptance test set for Repository interface.
// Only those which can pass this test are considered conformers of the interface.
//
// This test uses repository instances returned from newInitializedRepository.
// The function will be called multiple times.
// The test uses each instance however only one instance is used at a time.
// Therefore an invocation may invalidate all instances it has returned.
func TestRepository(t *testing.T, newInitializedRepository func() def.Repository) {
	t.Run("tasks can be added", func(t *testing.T) {
		TestRepository_tasks_can_be_added(t, newInitializedRepository())
	})

	t.Run("tasks can be updated", func(t *testing.T) {
		TestRepository_tasks_can_be_updated(t, newInitializedRepository())
	})

	t.Run("tasks meta data can be updated", func(t *testing.T) {
		TestRepository_meta_can_be_updated(t, newInitializedRepository())
	})
}
