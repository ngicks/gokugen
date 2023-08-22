package def

import (
	"testing"
	"time"

	"github.com/ngicks/und/option"
	"github.com/stretchr/testify/assert"
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

	scheduledTask = Task{
		Id:          NeverExistentId,
		WorkId:      "bar",
		State:       TaskScheduled,
		ScheduledAt: sampleDate,
		CreatedAt:   sampleDate,
	}
	cancelledTask = Task{
		Id:          NeverExistentId,
		WorkId:      "bar",
		State:       TaskCancelled,
		ScheduledAt: sampleDate,
		CreatedAt:   sampleDate,
		CancelledAt: option.Some(sampleDate.Add(time.Hour)),
	}
	dispatchedTask = Task{
		Id:           NeverExistentId,
		WorkId:       "bar",
		State:        TaskDispatched,
		ScheduledAt:  sampleDate,
		CreatedAt:    sampleDate,
		DispatchedAt: option.Some(sampleDate.Add(time.Hour)),
	}
	doneTask = Task{
		Id:           NeverExistentId,
		WorkId:       "bar",
		State:        TaskDone,
		ScheduledAt:  sampleDate,
		CreatedAt:    sampleDate,
		DispatchedAt: option.Some(sampleDate.Add(30 * time.Minute)),
		DoneAt:       option.Some(sampleDate.Add(time.Hour)),
	}
	errTask = Task{
		Id:           NeverExistentId,
		WorkId:       "bar",
		State:        TaskErr,
		ScheduledAt:  sampleDate,
		CreatedAt:    sampleDate,
		DispatchedAt: option.Some(sampleDate.Add(30 * time.Minute)),
		DoneAt:       option.Some(sampleDate.Add(time.Hour)),
		Err:          "fake error",
	}

	eachStateTasks = []Task{
		scheduledTask,
		cancelledTask,
		dispatchedTask,
		doneTask,
		errTask,
	}

	sampleTask = Task{
		Id:          NeverExistentId,
		WorkId:      "foo",
		State:       TaskScheduled,
		ScheduledAt: sampleDate.Add(time.Second),
		CreatedAt:   sampleDate,
	}

	fullySetTask = Task{
		Id:           NeverExistentId,
		WorkId:       "foo",
		Priority:     123,
		State:        TaskScheduled,
		Err:          "fake error",
		Param:        map[string]string{"foo": "bar"},
		Meta:         map[string]string{"baz": "qux"},
		ScheduledAt:  sampleDate.Add(time.Second),
		CreatedAt:    sampleDate,
		Deadline:     option.Some(sampleDate.Add(time.Minute)),
		CancelledAt:  option.Some(sampleDate.Add(30 * time.Second)),
		DispatchedAt: option.Some(sampleDate.Add(40 * time.Second)),
		DoneAt:       option.Some(sampleDate.Add(20 * time.Second)),
	}

	diff = Task{
		Id:           "2",
		WorkId:       "bar",
		Priority:     456,
		State:        TaskCancelled,
		Err:          "2 error 2",
		Param:        map[string]string{"foo": "barbar"},
		Meta:         map[string]string{"grault": "garply"},
		ScheduledAt:  sampleDate.Add(-1 * time.Second),
		CreatedAt:    sampleDate.Add(-5 * time.Minute),
		Deadline:     option.Some(sampleDate.Add(-time.Minute)),
		CancelledAt:  option.Some(sampleDate.Add(-30 * time.Second)),
		DispatchedAt: option.Some(sampleDate.Add(-40 * time.Second)),
		DoneAt:       option.Some(sampleDate.Add(-20 * time.Second)),
	}
)

func TestTask_Equal(t *testing.T) {
	assert := assert.New(t)

	assert.True(fullySetTask.Equal(fullySetTask))
	assert.True(diff.Equal(diff))

	for i := 1; i <= 0b1111111111111; i++ {
		cloned := fullySetTask.Clone()
		if i>>0&1 != 0 {
			cloned.Id = diff.Id
		}
		if i>>1&1 != 0 {
			cloned.WorkId = diff.WorkId
		}
		if i>>2&1 != 0 {
			cloned.Priority = diff.Priority
		}
		if i>>3&1 != 0 {
			cloned.State = diff.State
		}
		if i>>4&1 != 0 {
			cloned.Err = diff.Err
		}
		if i>>5&1 != 0 {
			cloned.Param = diff.Param
		}
		if i>>6&1 != 0 {
			cloned.Meta = diff.Meta
		}
		if i>>7&1 != 0 {
			cloned.ScheduledAt = diff.ScheduledAt
		}
		if i>>8&1 != 0 {
			cloned.CreatedAt = diff.CreatedAt
		}
		if i>>9&1 != 0 {
			cloned.Deadline = diff.Deadline
		}
		if i>>10&1 != 0 {
			cloned.CancelledAt = diff.CancelledAt
		}
		if i>>11&1 != 0 {
			cloned.DispatchedAt = diff.DispatchedAt
		}
		if i>>12&1 != 0 {
			cloned.DoneAt = diff.DoneAt
		}

		assert.False(cloned.Equal(fullySetTask), "bit flag = %b, task = %+#v", i, cloned)
	}
}

func TestTask_IsValid(t *testing.T) {
	assert := assert.New(t)

	assert.True(sampleTask.IsValid())

	cloned := sampleTask.Clone()
	assert.Equal(0, len(cloned.ReportInvalidity()))

	for i := 1; i <= 0b11111; i++ {
		if i>>0&1 != 0 {
			cloned.Id = ""
		}
		if i>>1&1 != 0 {
			cloned.WorkId = ""
		}
		if i>>2&1 != 0 {
			cloned.State = ""
		}
		if i>>3&1 != 0 {
			cloned.ScheduledAt = time.Time{}
		}
		if i>>4&1 != 0 {
			cloned.CreatedAt = time.Time{}
		}
		assert.False(cloned.IsValid())
		assert.Greater(len(cloned.ReportInvalidity()), 0)
	}
}
