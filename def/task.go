package def

import (
	"fmt"
	"time"

	"github.com/ngicks/gokugen/scheduler/util"
	"github.com/ngicks/und/option"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type State string

const (
	Scheduled  State = "scheduled"
	Dispatched State = "dispatched"
	Cancelled  State = "cancelled"
	Done       State = "done"
	Err        State = "err"
)

var states = [...]string{
	string(Scheduled),
	string(Dispatched),
	string(Cancelled),
	string(Done),
	string(Err),
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
	Param        map[string]string        `json:"param"`
	Priority     int                      `json:"priority"`
	State        State                    `json:"state"`
	ScheduledAt  time.Time                `json:"scheduled_at"`
	CreatedAt    time.Time                `json:"created_at"`
	CancelledAt  option.Option[time.Time] `json:"cancelled_at"`
	DispatchedAt option.Option[time.Time] `json:"dispatched_at"`
	DoneAt       option.Option[time.Time] `json:"done_at"`
	Err          string                   `json:"err"`
	Meta         map[string]string        `json:"meta"`
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

// TruncTime trims time.Time fields down to milliseconds precision.
// TruncTime may be useful when it is sending this task to systems
// which are not capable of working on micro or finer time granularity.
func (t Task) TruncTime() Task {
	t = t.Clone()
	t.ScheduledAt = util.DropMicros(t.ScheduledAt)
	t.CreatedAt = util.DropMicros(t.CreatedAt)
	t.CancelledAt = t.CancelledAt.Map(util.DropMicros)
	t.DispatchedAt = t.DispatchedAt.Map(util.DropMicros)
	t.DoneAt = t.DoneAt.Map(util.DropMicros)
	return t
}

func (t Task) Less(j Task) bool {
	if t.CancelledAt.IsSome() {
		return true
	}

	if !t.ScheduledAt.Equal(j.ScheduledAt) {
		return t.ScheduledAt.Before(j.ScheduledAt)
	}

	return t.Priority > j.Priority
}

func (t Task) Update(param TaskParam, ignoreMicro bool) Task {
	t = t.Clone()

	if param.WorkId.IsSome() {
		t.WorkId = param.WorkId.Value()
	}
	if param.Param.IsSome() {
		paramUpdater := param.Param.Value()
		if t.Param == nil && len(paramUpdater) > 0 {
			t.Param = map[string]string{}
		}
		for k, v := range paramUpdater {
			switch v.OpTy {
			case MapUpdateRemoveKey:
				delete(t.Param, k)
			case MapUpdateSetKey:
				t.Param[k] = v.Val
			}
		}
	}
	if param.Priority.IsSome() {
		t.Priority = param.Priority.Value()
	}
	if param.State.IsSome() {
		t.State = param.State.Value()
	}
	if param.ScheduledAt.IsSome() {
		t.ScheduledAt = param.ScheduledAt.Value()
	}
	if param.CreatedAt.IsSome() {
		t.CreatedAt = param.CreatedAt.Value()
	}
	if param.CancelledAt.IsSome() {
		t.CancelledAt = param.CancelledAt.Value()
	}
	if param.DispatchedAt.IsSome() {
		t.DispatchedAt = param.DispatchedAt.Value()
	}
	if param.DoneAt.IsSome() {
		t.DoneAt = param.DoneAt.Value()
	}
	if param.Err.IsSome() {
		t.Err = param.Err.Value()
	}
	if param.Meta.IsSome() {
		metaUpdater := param.Meta.Value()
		if t.Meta == nil && len(metaUpdater) > 0 {
			t.Meta = map[string]string{}
		}
		for k, v := range metaUpdater {
			switch v.OpTy {
			case MapUpdateRemoveKey:
				delete(t.Meta, k)
			case MapUpdateSetKey:
				t.Meta[k] = v.Val
			}
		}
	}

	if ignoreMicro {
		t = t.TruncTime()
	}

	return t
}
