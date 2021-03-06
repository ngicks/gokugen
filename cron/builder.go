package cron

import (
	"time"
)

// Builder helps easy building of Row.
type Builder struct {
	cron Row
}

// Minute overwrites Minute field.
// Passing zero arguments set it as wildcard.
func (b Builder) Minute(in ...uint8) Builder {
	minutes := Minutes([]Minute{})
	for _, m := range in {
		minutes = append(minutes, Minute(int(m)))
	}
	b.cron.Minute = &minutes
	return b
}

// Hour overwrites Hour field.
// Passing zero arguments set it as wildcard.
func (b Builder) Hour(in ...uint8) Builder {
	hours := Hours([]Hour{})
	for _, m := range in {
		hours = append(hours, Hour(int(m)))
	}
	b.cron.Hour = &hours
	return b
}

// Day overwrites Day field.
// Passing zero arguments set it as wildcard.
func (b Builder) Day(in ...uint8) Builder {
	days := Days([]Day{})
	for _, m := range in {
		days = append(days, Day(int(m)))
	}
	b.cron.Day = &days
	return b
}

// Month overwrites Month field.
// Passing zero arguments set it as wildcard.
func (b Builder) Month(in ...time.Month) Builder {
	months := Months([]time.Month{})
	for _, m := range in {
		months = append(months, m)
	}
	b.cron.Month = &months
	return b
}

// WeekDay overwrites WeekDay field.
// Passing zero arguments set it as wildcard.
func (b Builder) WeekDay(in ...time.Weekday) Builder {
	weekdays := Weekdays([]time.Weekday{})
	for _, w := range in {
		weekdays = append(weekdays, w)
	}
	b.cron.Weekday = &weekdays
	return b
}

// Command overwrites Command field.
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

func (b Builder) Yearly(month []time.Month, day, hour, minute []uint8) Builder {
	return b.clearTime().Month(month...).Day(day...).Hour(hour...).Minute(minute...)
}

func (b Builder) Monthly(day, hour, minute []uint8) Builder {
	return b.clearTime().Day(day...).Hour(hour...).Minute(minute...)
}

func (b Builder) Weekly(weekDay []time.Weekday, hour, minute []uint8) Builder {
	return b.clearTime().WeekDay(weekDay...).Hour(hour...).Minute(minute...)
}

func (b Builder) Daily(hour, minute []uint8) Builder {
	return b.clearTime().Hour(hour...).Minute(minute...)
}

func (b Builder) Hourly(minute []uint8) Builder {
	return b.clearTime().Minute(minute...)
}

// Reboot clears all time value fields.
// This will make row as a once task.
func (b Builder) Reboot() Builder {
	return b.clearTime()
}

// Build builds Row.
// No validation happens.
func (b Builder) Build() Row {
	return b.cron
}
