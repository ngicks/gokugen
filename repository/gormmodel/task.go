package gormmodel

import (
	"time"

	"github.com/ngicks/gokugen/scheduler"
)

type Task struct {
	Id           string     `json:"id" gorm:"primaryKey;not null"`
	WorkId       string     `json:"work_id" gorm:"not null"`
	Param        string     `json:"param"`
	ScheduledAt  time.Time  `json:"scheduled_at" gorm:"not null;index:sched;sort:asc"`
	Priority     int        `json:"priority" gorm:"not null;index:sched;sort:desc"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;autoCreateTime:milli"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty"`
	DispatchedAt *time.Time `json:"dispatched_at,omitempty"`
	DoneAt       *time.Time `json:"done_at,omitempty"`
	Err          string     `json:"err"`
	Meta         Meta       `json:"meta" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UpdatedAt    int64      `gorm:"autoUpdateTime:milli"`
}

func FromTask(t scheduler.Task) Task {
	return Task{
		Id:           t.Id,
		WorkId:       t.WorkId,
		Param:        string(t.Param),
		ScheduledAt:  t.ScheduledAt,
		Priority:     t.Priority,
		CreatedAt:    t.CreatedAt,
		CancelledAt:  t.CancelledAt,
		DispatchedAt: t.DispatchedAt,
		DoneAt:       t.DoneAt,
		Err:          t.Err,
		Meta:         FromMeta(t.Id, t.Meta),
	}
}

func (t Task) ToTask() scheduler.Task {
	return scheduler.Task{
		Id:           t.Id,
		WorkId:       t.WorkId,
		Param:        []byte(t.Param),
		ScheduledAt:  t.ScheduledAt,
		Priority:     t.Priority,
		CreatedAt:    t.CreatedAt,
		CancelledAt:  t.CancelledAt,
		DispatchedAt: t.DispatchedAt,
		DoneAt:       t.DoneAt,
		Err:          t.Err,
		Meta:         Meta(t.Meta).ToMeta(),
	}
}

type TaskMatcher struct {
	Task
	Priority *int
}

func FromTaskMatcher(m scheduler.TaskMatcher) TaskMatcher {
	return TaskMatcher{
		Task:     FromTask(m.Task),
		Priority: m.Priority,
	}
}

func (t TaskMatcher) VisitNonZero(visitor TaskVisitor) {
	if t.Id != "" && visitor.Id != nil {
		visitor.Id(t.Id)
	}
	if t.WorkId != "" && visitor.WorkId != nil {
		visitor.WorkId(t.WorkId)
	}
	if t.Param != "" && visitor.Param != nil {
		visitor.Param(t.Param)
	}
	if !t.ScheduledAt.IsZero() && visitor.ScheduledAt != nil {
		visitor.ScheduledAt(t.ScheduledAt)
	}
	if t.Priority != nil && visitor.Priority != nil {
		visitor.Priority(t.Priority)
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
	if t.UpdatedAt != 0 && visitor.UpdatedAt != nil {
		visitor.UpdatedAt(t.UpdatedAt)
	}
}

type TaskVisitor struct {
	Id           func(v string)
	WorkId       func(v string)
	Param        func(v string)
	ScheduledAt  func(v time.Time)
	Priority     func(v *int)
	CreatedAt    func(v time.Time)
	CancelledAt  func(v *time.Time)
	DispatchedAt func(v *time.Time)
	DoneAt       func(v *time.Time)
	Err          func(v string)
	Meta         func(v Meta)
	UpdatedAt    func(v int64)
}
