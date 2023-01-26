package scheduler

import (
	"bytes"
	"time"

	"github.com/ngicks/gokugen/scheduler/util"
)

type Task struct {
	Id           string            `json:"id"`      // Id is an id of the task.
	WorkId       string            `json:"work_id"` // WorkId is work function id.
	Param        []byte            `json:"param"`
	Priority     int               `json:"priority"`
	ScheduledAt  time.Time         `json:"scheduled_at"`
	CreatedAt    time.Time         `json:"created_at"`
	CancelledAt  *time.Time        `json:"cancelled_at,omitempty"`
	DispatchedAt *time.Time        `json:"dispatched_at,omitempty"`
	DoneAt       *time.Time        `json:"done_at,omitempty"`
	Err          string            `json:"err"`
	Meta         map[string][]byte `json:"meta"`
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
	if param.Meta != nil {
		if t.Meta == nil {
			t.Meta = param.Meta
		} else {
			for k, v := range param.Meta {
				t.Meta[k] = v
			}
		}
	}

	return t
}

func (t Task) Equal(other Task) bool {
	if t.Id != other.Id {
		return false
	}

	mapEqual := func(l, r map[string][]byte) bool {
		if len(l) != len(r) {
			return false
		}
		for k, v := range l {
			if !bytes.Equal(v, r[k]) {
				return false
			}
		}
		return true
	}

	return (t.WorkId == other.WorkId &&
		bytes.Equal(t.Param, other.Param) &&
		t.Priority == other.Priority &&
		t.ScheduledAt.Equal(other.ScheduledAt) &&
		t.CreatedAt.Equal(other.CreatedAt) &&
		util.TimePointerEqual(t.CancelledAt, other.CancelledAt, false) &&
		util.TimePointerEqual(t.DispatchedAt, other.DispatchedAt, false) &&
		util.TimePointerEqual(t.DoneAt, other.DoneAt, false) &&
		t.Err == other.Err &&
		mapEqual(t.Meta, other.Meta))
}

func cloneMeta(meta map[string][]byte) map[string][]byte {
	cloned := make(map[string][]byte, len(meta))
	for k, v := range meta {
		cloned[k] = v
	}
	return cloned
}

func (t Task) ToParam() TaskParam {
	p := t.Priority
	return TaskParam{
		ScheduledAt: t.ScheduledAt,
		WorkId:      t.WorkId,
		Param:       t.Param,
		Priority:    &p,
		Meta:        t.Meta,
	}
}

// Serializable is serializable part
type Serializable struct {
	Id     string            `json:"id"`
	WorkId string            `json:"work_id"`
	Param  []byte            `json:"param"`
	Meta   map[string][]byte `json:"meta"`
}

type TaskParam struct {
	// scheduled time.
	// The zero value of ScheduledAt is considered to be a non-initialized field in AddTask context,
	// and not to be a update target in Update context.
	ScheduledAt time.Time
	WorkId      string
	Param       []byte
	Priority    *int
	Meta        map[string][]byte
}

func cloneSlice[T any](s []T) []T {
	cloned := make([]T, len(s))
	copy(cloned, s)
	return cloned
}

func (p TaskParam) Clone() TaskParam {
	var priority int
	if p.Priority != nil {
		priority = *p.Priority
	}
	return TaskParam{
		ScheduledAt: p.ScheduledAt,
		WorkId:      p.WorkId,
		Param:       cloneSlice(p.Param),
		Priority:    &priority,
		Meta:        cloneMeta(p.Meta),
	}
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
		Meta:        p.Meta,
	}
}
