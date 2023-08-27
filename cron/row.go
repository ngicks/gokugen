package cron

import (
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
	"github.com/robfig/cron/v3"
)

type CronRowRaw struct {
	Param    def.TaskUpdateParam
	Schedule string
}

func (r CronRowRaw) Parse() (CronRow, error) {
	sched, err := cron.ParseStandard(r.Schedule)
	if err != nil {
		return CronRow{}, err
	}
	return CronRow{
		Param:    r.Param.Clone(),
		Schedule: sched,
	}, nil
}

type CronRow struct {
	Param    def.TaskUpdateParam
	Schedule cron.Schedule
}

func (r CronRow) Next(prev time.Time) def.TaskUpdateParam {
	return r.Param.Update(def.TaskUpdateParam{ScheduledAt: option.Some(r.Schedule.Next(prev))})
}
