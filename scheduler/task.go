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
	Meta         map[string]string `json:"meta"`
}

func (t Task) VisitNonZero(visitor TaskVisitor) {

	if t.Id != "" && visitor.Id != nil {
		visitor.Id(t.Id)
	}
	if t.WorkId != "" && visitor.WorkId != nil {
		visitor.WorkId(t.WorkId)
	}
	if t.Param != nil && visitor.Param != nil {
		visitor.Param(t.Param)
	}
	if t.Priority != 0 && visitor.Priority != nil {
		visitor.Priority(t.Priority)
	}
	if !t.ScheduledAt.IsZero() && visitor.ScheduledAt != nil {
		visitor.ScheduledAt(t.ScheduledAt)
	}
	if !t.CreatedAt.IsZero() && visitor.CreatedAt != nil {
		visitor.CreatedAt(t.CreatedAt)
	}
	if t.CancelledAt != nil && visitor.CancelledAt != nil {
		visitor.CancelledAt(t.CancelledAt)
	}
	if t.DispatchedAt != nil && visitor.DispatchedAt != nil {
		visitor.DispatchedAt(t.DispatchedAt)
	}
	if t.DoneAt != nil && visitor.DoneAt != nil {
		visitor.DoneAt(t.DoneAt)
	}
	if t.Err != "" && visitor.Err != nil {
		visitor.Err(t.Err)
	}
	if t.Meta != nil && visitor.Meta != nil {
		visitor.Meta(t.Meta)
	}
}

type TaskVisitor struct {
	Id           func(string)
	WorkId       func(string)
	Param        func([]byte)
	Priority     func(int)
	ScheduledAt  func(time.Time)
	CreatedAt    func(time.Time)
	CancelledAt  func(*time.Time)
	DispatchedAt func(*time.Time)
	DoneAt       func(*time.Time)
	Err          func(string)
	Meta         func(map[string]string)
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
	if t.CancelledAt != nil {
		return true
	}

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
		t.Meta = cloneMeta(param.Meta)
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
		t.Err == other.Err &&
		metaEqual(t.Meta, other.Meta))
}

func (t Task) Match(u TaskMatcher) bool {
	if u.WorkId != "" && t.WorkId != u.WorkId ||
		u.Param != nil && !bytes.Equal(t.Param, u.Param) ||
		u.Priority != nil && t.Priority != *u.Priority ||
		!u.ScheduledAt.IsZero() && !t.ScheduledAt.Equal(u.ScheduledAt) ||
		!u.CreatedAt.IsZero() && !t.CreatedAt.Equal(u.CreatedAt) ||
		u.CancelledAt != nil && !util.TimePointerEqual(t.CancelledAt, u.CancelledAt, false) ||
		u.DispatchedAt != nil && !util.TimePointerEqual(t.DispatchedAt, u.DispatchedAt, false) ||
		u.DoneAt != nil && !util.TimePointerEqual(t.DoneAt, u.DoneAt, false) ||
		u.Err != "" && t.Err != u.Err ||
		u.Meta != nil && !metaEqual(t.Meta, u.Meta) {
		return false
	}
	return true
}

func cloneMeta(meta map[string]string) map[string]string {
	cloned := make(map[string]string, len(meta))
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

type TaskParam struct {
	// scheduled time.
	// The zero value of ScheduledAt is considered to be a non-initialized field in AddTask context,
	// and not to be a update target in Update context.
	ScheduledAt time.Time
	WorkId      string
	Param       []byte
	Priority    *int
	Meta        map[string]string
}

// HasOnlyMeta reports if p only has Meta field.
// It returns true if all fields but Meta is zero value, false otherwise.
// This is used mainly in TaskRepository implementations.
func (p TaskParam) HasOnlyMeta() bool {
	return p.ScheduledAt.IsZero() &&
		p.WorkId == "" &&
		p.Param == nil &&
		p.Priority == nil &&
		p.Meta != nil // not zero!
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
		Param:       p.Param,
		Priority:    priority,
		Meta:        p.Meta,
	}
}

func metaEqual(l, r map[string]string) bool {
	if len(l) != len(r) {
		return false
	}
	for k, v := range l {
		if v != r[k] {
			return false
		}
	}
	return true
}

type TaskMatcher struct {
	Task
	Priority *int
}

type TaskMatcherVisitor struct {
	TaskVisitor
	Priority func(p *int)
}

func (m TaskMatcher) VisitNonZero(visitor TaskMatcherVisitor) {
	visitor.TaskVisitor.Priority = nil

	m.Task.VisitNonZero(visitor.TaskVisitor)
	if m.Priority != nil && visitor.Priority != nil {
		visitor.Priority(m.Priority)
	}

}
