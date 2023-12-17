package cron

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
)

const (
	MetaKeyScheduleHash = "ngicks.ScheduleHash"
)

type serializable struct {
	WorkId   string `json:"work_id"`
	Priority int    `json:"priority"`
	Param    string `json:"param"`
	Meta     string `json:"meta"`
}

func paramToSerializable(p def.TaskUpdateParam) serializable {
	param, meta := `{}`, `{}`

	if v := p.Param.Value(); len(v) > 0 {
		paramBin, _ := json.Marshal(v)
		param = string(paramBin)
	}
	if v := p.Meta.Value(); len(v) > 0 {
		metaBin, _ := json.Marshal(v)
		meta = string(metaBin)
	}

	return serializable{
		WorkId:   p.WorkId.Value(),
		Priority: p.Priority.Value(),
		Param:    param,
		Meta:     meta,
	}
}

type Schedule interface {
	// ScheduleHash returns identity value for scheduling.
	//
	// Normally this should return string representing cron expression,
	// however it can be anything.
	ScheduleHash() string
	Next(prev time.Time) def.TaskUpdateParam
}

type Entry struct {
	mu       sync.Mutex
	prev     time.Time
	schedule Schedule
}

func NewEntry(t time.Time, schedule Schedule) *Entry {
	return &Entry{
		prev:     t,
		schedule: schedule,
	}
}

func (e *Entry) next() def.TaskUpdateParam {
	next := e.schedule.Next(e.prev)
	next.Meta = next.Meta.
		Or(option.Some(make(map[string]string))).
		Map(func(v map[string]string) map[string]string {
			v[MetaKeyScheduleHash] = e.schedule.ScheduleHash()
			return v
		})
	return next
}

// Params returns next schedule without advancing e to next time.
func (e *Entry) Param() def.TaskUpdateParam {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.next()
}

// Next returns next schedule and advances e to next time.
func (e *Entry) Next() def.TaskUpdateParam {
	e.mu.Lock()
	defer e.mu.Unlock()
	next := e.next()
	e.prev = next.ScheduledAt.Value()
	return next
}
