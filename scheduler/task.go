package scheduler

import (
	"bytes"
	"time"

	"github.com/ngicks/gokugen/scheduler/util"
)

type Task struct {
	Id           string     `json:"id"`      // Id is an id of the task.
	WorkId       string     `json:"work_id"` // WorkId is work function id.
	Param        []byte     `json:"param"`
	Priority     int        `json:"priority"`
	ScheduledAt  time.Time  `json:"scheduled_at"`
	CreatedAt    time.Time  `json:"created_at"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"`
	DispatchedAt *time.Time `json:"dispatched_at,omitempty"`
	DoneAt       *time.Time `json:"done_at,omitempty"`
	Err          string     `json:"err"`
}

func (t Task) IsInitialized() bool {
	return t.Id != "" &&
		t.WorkId != "" &&
		!t.ScheduledAt.IsZero()
}

func (t Task) DropMicros() Task {
	t.ScheduledAt = util.DropMicros(t.ScheduledAt)
	t.CreatedAt = util.DropMicros(t.CreatedAt)
	t.CancelledAt = util.DropMicrosPointer(t.CancelledAt)
	t.DispatchedAt = util.DropMicrosPointer(t.DispatchedAt)
	t.DoneAt = util.DropMicrosPointer(t.DoneAt)
	return t
}

func (t Task) Less(j Task) bool {
	if !t.ScheduledAt.Equal(j.ScheduledAt) {
		return t.ScheduledAt.Before(j.ScheduledAt)
	}
	return t.Priority > j.Priority
}

func (t Task) Update(param TaskParam, ignoreMicroSecs bool) Task {
	if !param.ScheduledAt.IsZero() {
		if ignoreMicroSecs {
			t.ScheduledAt = util.DropMicros(param.ScheduledAt)
		} else {
			t.ScheduledAt = param.ScheduledAt
		}
	}
	if param.WorkId != "" {
		t.WorkId = param.WorkId
	}
	if param.Param != nil {
		t.Param = param.Param
	}
	if param.Priority != nil {
		t.Priority = *param.Priority
	}

	return t
}

func (t Task) Equal(other Task) bool {
	if t.Id != other.Id {
		return false
	}

	return (t.WorkId == other.WorkId &&
		bytes.Equal(t.Param, other.Param) &&
		t.Priority == other.Priority &&
		t.ScheduledAt.Equal(other.ScheduledAt) &&
		t.CreatedAt.Equal(other.CreatedAt) &&
		util.TimePointerEqual(t.CancelledAt, other.CancelledAt, false) &&
		util.TimePointerEqual(t.DispatchedAt, other.DispatchedAt, false) &&
		util.TimePointerEqual(t.DoneAt, other.DoneAt, false) &&
		t.Err == other.Err)
}

func (t Task) ToParam() TaskParam {
	p := t.Priority
	return TaskParam{
		ScheduledAt: t.ScheduledAt,
		WorkId:      t.WorkId,
		Param:       t.Param,
		Priority:    &p,
	}
}

// Serializable is serializable part
type Serializable struct {
	Id     string `json:"id"`
	WorkId string `json:"work_id"`
	Param  []byte `json:"param"`
}

type TaskParam struct {
	// scheduled time.
	// The zero value of ScheduledAt is considered to be non-initialized in AddTask context,
	// and to be not a update target in Update context.
	ScheduledAt time.Time
	WorkId      string
	Param       []byte
	Priority    *int
}

func (p TaskParam) IsInitialized() bool {
	return !p.ScheduledAt.IsZero() && p.WorkId != ""
}

func (p TaskParam) ToTask(ignoreMicros bool) Task {
	var param []byte
	if p.Param != nil {
		param = make([]byte, len(p.Param))
		copy(param, p.Param)
	}

	var scheduledAt time.Time
	if ignoreMicros {
		scheduledAt = util.DropMicros(p.ScheduledAt)
	} else {
		scheduledAt = p.ScheduledAt
	}

	var priority int
	if p.Priority != nil {
		priority = *p.Priority
	}

	return Task{
		ScheduledAt: scheduledAt,
		WorkId:      p.WorkId,
		Param:       param,
		Priority:    priority,
	}
}
