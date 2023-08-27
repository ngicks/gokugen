package cron

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ngicks/gokugen/def"
)

type serializable struct {
	WorkId   string `json:"work_id"`
	Priority int    `json:"priority"`
	Param    string `json:"param"`
	Meta     string `json:"meta"`
}

func paramToSerializable(p def.TaskUpdateParam) serializable {
	param, meta := `{}`, `{}`

	if v := p.Param.Value(); v != nil {
		paramBin, _ := json.Marshal(v)
		param = string(paramBin)
	}
	if v := p.Meta.Value(); v != nil {
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
	Next(prev time.Time) def.TaskUpdateParam
}

type Entry struct {
	mu       sync.Mutex
	prev     time.Time
	schedule Schedule
}

func From(t time.Time, schedule Schedule) *Entry {
	return &Entry{
		prev:     t,
		schedule: schedule,
	}
}

func (e *Entry) Next() def.TaskUpdateParam {
	e.mu.Lock()
	defer e.mu.Unlock()
	next := e.schedule.Next(e.prev)
	e.prev = next.ScheduledAt.Value()
	return next
}
