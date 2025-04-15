package def

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/ngicks/gokugen/def/util"
	"github.com/ngicks/und/option"
)

type State string

const (
	TaskScheduled  State = "scheduled"
	TaskDispatched State = "dispatched"
	TaskCancelled  State = "cancelled"
	TaskDone       State = "done"
	TaskErr        State = "err"
)

var states = [...]string{
	string(TaskScheduled),
	string(TaskDispatched),
	string(TaskCancelled),
	string(TaskDone),
	string(TaskErr),
}

func IsState(s string) bool {
	return slices.Contains(states[:], s)
}

// GetStates returns all variants of State as a string array.
func GetStates() []string {
	return states[:]
}

type Task struct {
	Id           string                   `json:"id"`      // Id is an id of the task.
	WorkId       string                   `json:"work_id"` // WorkId is work function id.
	Priority     int                      `json:"priority"`
	State        State                    `json:"state"`
	Err          string                   `json:"err"`
	Param        map[string]string        `json:"param"`
	Meta         map[string]string        `json:"meta"`
	ScheduledAt  time.Time                `json:"scheduled_at"`
	CreatedAt    time.Time                `json:"created_at"`
	Deadline     option.Option[time.Time] `json:"deadline"`
	CancelledAt  option.Option[time.Time] `json:"cancelled_at"`
	DispatchedAt option.Option[time.Time] `json:"dispatched_at"`
	DoneAt       option.Option[time.Time] `json:"done_at"`
}

func (t Task) Equal(u Task) bool {
	return fromTaskToComparable(t) == fromTaskToComparable(u) &&
		maps.Equal(t.Param, u.Param) &&
		maps.Equal(t.Meta, u.Meta) &&
		t.ScheduledAt.Equal(u.ScheduledAt) &&
		t.CreatedAt.Equal(u.CreatedAt) &&
		option.Equal(t.Deadline, u.Deadline) &&
		option.Equal(t.CancelledAt, u.CancelledAt) &&
		option.Equal(t.DispatchedAt, u.DispatchedAt) &&
		option.Equal(t.DoneAt, u.DoneAt)
}

type taskComparable struct {
	Id       string
	WorkId   string
	Priority int
	State    State
	Err      string
}

func fromTaskToComparable(t Task) taskComparable {
	var c taskComparable

	c.Id = t.Id
	c.WorkId = t.WorkId
	c.Priority = t.Priority
	c.State = t.State
	c.Err = t.Err

	return c
}

func (t Task) IsValid() bool {
	return t.Id != "" &&
		t.WorkId != "" &&
		IsState(string(t.State)) &&
		!t.ScheduledAt.IsZero() &&
		!t.CreatedAt.IsZero()
}

type InvalidField struct {
	Name   string
	Reason string
}

func (t Task) ReportInvalidity() []InvalidField {
	var result []InvalidField

	if t.Id == "" {
		result = append(result, InvalidField{Name: "id", Reason: "empty id"})
	}
	if t.WorkId == "" {
		result = append(result, InvalidField{Name: "work_id", Reason: "empty work_id"})
	}
	if !IsState(string(t.State)) {
		result = append(
			result,
			InvalidField{
				Name: "state",
				Reason: fmt.Sprintf(
					"invalid state. state is '%s' while it must be one of %s",
					t.State, GetStates(),
				),
			},
		)
	}
	if t.ScheduledAt.IsZero() {
		result = append(
			result,
			InvalidField{
				Name:   "scheduled_at",
				Reason: "zero value is not allowed for this field",
			},
		)
	}
	if t.CreatedAt.IsZero() {
		result = append(
			result,
			InvalidField{
				Name:   "created_at",
				Reason: "zero value is not allowed for this field",
			},
		)
	}

	return result
}

func (t Task) Clone() Task {
	t.Param = maps.Clone(t.Param)
	t.Meta = maps.Clone(t.Meta)
	return t
}

// NormalizeTime trims time.Time fields down to milliseconds precision.
// NormalizeTime may be useful when it is sending this task to systems
// which are not capable of working on micro or finer time granularity.
func (t Task) NormalizeTime() Task {
	t = t.Clone()
	t.ScheduledAt = NormalizeTime(t.ScheduledAt)
	t.CreatedAt = NormalizeTime(t.CreatedAt)
	t.Deadline = t.Deadline.Map(NormalizeTime)
	t.CancelledAt = t.CancelledAt.Map(NormalizeTime)
	t.DispatchedAt = t.DispatchedAt.Map(NormalizeTime)
	t.DoneAt = t.DoneAt.Map(NormalizeTime)
	return t
}

func (t Task) Less(j Task) bool {
	if t.CancelledAt.IsSome() {
		return true
	}

	if !t.ScheduledAt.Equal(j.ScheduledAt) {
		return t.ScheduledAt.Before(j.ScheduledAt)
	}

	if t.Priority != j.Priority {
		return t.Priority > j.Priority
	}

	return t.CreatedAt.Before(j.CreatedAt)
}

func (t Task) Update(param TaskUpdateParam) Task {
	assignIfSome(&t.WorkId, param.WorkId, nil, nil)
	assignIfSome(
		&t.Param,
		param.Param,
		func(v map[string]string) bool { return v == nil },
		func() map[string]string { return map[string]string{} },
	)
	assignIfSome(&t.Priority, param.Priority, nil, nil)
	assignIfSome(&t.ScheduledAt, param.ScheduledAt, nil, nil)
	assignIfSome(&t.Deadline, param.Deadline, nil, nil)
	assignIfSome(
		&t.Meta,
		param.Meta,
		func(v map[string]string) bool { return v == nil },
		func() map[string]string { return map[string]string{} },
	)

	t = t.NormalizeTime()

	return t
}

func NormalizeTime(t time.Time) time.Time {
	return util.DropMicros(t).In(time.UTC)
}

func assignIfSome[T any](
	v *T,
	updater option.Option[T],
	isZero func(v T) bool,
	getDefaultValue func() T,
) {
	if updater.IsSome() {
		updaterValue := updater.Value()
		if isZero != nil && isZero(updaterValue) {
			*v = getDefaultValue()
		} else {
			*v = updaterValue
		}
	}
}
