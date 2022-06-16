package cron

import (
	"errors"
	"time"
)

var (
	ErrNilWork = errors.New("nil work")
)

type Builder struct {
	cron Row
}

func (b Builder) Minute(in1 uint8, in2 ...uint8) Builder {
	minutes := Minutes([]Minute{Minute(int(in1))})
	for _, m := range in2 {
		minutes = append(minutes, Minute(int(m)))
	}
	b.cron.Minute = &minutes
	return b
}

func (b Builder) Hour(in1 uint8, in2 ...uint8) Builder {
	hours := Hours([]Hour{Hour(int(in1))})
	for _, m := range in2 {
		hours = append(hours, Hour(int(m)))
	}
	b.cron.Hour = &hours
	return b
}

func (b Builder) Day(in1 uint8, in2 ...uint8) Builder {
	days := Days([]Day{Day(int(in1))})
	for _, m := range in2 {
		days = append(days, Day(int(m)))
	}
	b.cron.Day = &days
	return b
}

func (b Builder) Month(in1 time.Month, in2 ...time.Month) Builder {
	months := Months([]time.Month{time.Month(int(in1))})
	for _, m := range in2 {
		months = append(months, m)
	}
	b.cron.Month = &months
	return b
}

func (b Builder) WeekDay(in1 time.Weekday, in2 ...time.Weekday) Builder {
	weekdays := Weekdays([]time.Weekday{time.Weekday(int(in1))})
	for _, w := range in2 {
		weekdays = append(weekdays, w)
	}
	b.cron.Weekday = &weekdays
	return b
}

func (b Builder) Command(in []string) Builder {
	b.cron.Command = in
	return b
}

func (b Builder) clearTime() Builder {
	newRow := Row{}
	newRow.Command = b.cron.Command
	b.cron = newRow
	return b
}

func (b Builder) Yearly(month time.Month, day, hour, minute uint8) Builder {
	return b.clearTime().Month(month).Day(day).Hour(hour).Minute(minute)
}

func (b Builder) Monthly(day, hour, minute uint8) Builder {
	return b.clearTime().Day(day).Hour(hour).Minute(minute)
}

func (b Builder) Weekly(weekDay time.Weekday, hour, minute uint8) Builder {
	return b.clearTime().WeekDay(weekDay).Hour(hour).Minute(minute)
}

func (b Builder) Daily(hour, minute uint8) Builder {
	return b.clearTime().Hour(hour).Minute(minute)
}

func (b Builder) Hourly(minute uint8) Builder {
	return b.clearTime().Minute(minute)
}

func (b Builder) Reboot() Builder {
	return b.clearTime()
}

func (b Builder) Build() (Row, error) {
	if b.cron.Command == nil {
		return b.cron, ErrNilWork
	}
	return b.cron, nil
}

func (b Builder) MustBuild() Row {
	row, err := b.Build()
	if err != nil {
		panic(err)
	}
	return row
}
