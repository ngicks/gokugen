package gormtask

import (
	"encoding/json"
	"time"

	"github.com/ngicks/gokugen/scheduler"
	"gorm.io/gorm"
)

type GormTask struct {
	// Id is an id of the task.
	Id string `json:"id" gorm:"primaryKey,not null"`
	// WorkId is work function id.
	WorkId       string         `json:"work_id" gorm:"not null"`
	Param        string         `json:"param"`
	ScheduledAt  time.Time      `json:"scheduled_at" gorm:"not null,index:sched,sort:asc"`
	Priority     int            `json:"priority" gorm:"not null,index:sched,sort:desc"`
	CreatedAt    time.Time      `json:"created_at" gorm:"not null,autoCreateTime:milli"`
	CancelledAt  *time.Time     `json:"cancelled_at,omitempty"`
	DispatchedAt *time.Time     `json:"dispatched_at,omitempty"`
	DoneAt       *time.Time     `json:"done_at,omitempty"`
	Err          string         `json:"err"`
	Meta         []byte         `json:"meta"`
	UpdatedAt    int64          `gorm:"autoUpdateTime:milli"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index:deleted"`
}

func FromTask(t scheduler.Task) GormTask {
	marshalledMeta, err := json.Marshal(t.Meta)
	if err != nil {
		panic(err)
	}

	return GormTask{
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
		Meta:         marshalledMeta,
	}
}

func (t GormTask) ToTask() scheduler.Task {
	meta := make(map[string]string)
	if err := json.Unmarshal(t.Meta, &meta); err != nil {
		panic(err)
	}

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
		Meta:         meta,
	}
}
