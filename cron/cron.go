package cron

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMalformed = errors.New("malformed cron row")
	ErrNoMatch   = errors.New("no match")
)

type WorkFn = func(targeted, current time.Time, repeat int) (any, error)

type CronTab []Row

// Row is representation of a crontab row. It may behave differently, but it should do similarly.
// If each of time field has no value set (set to nil) or empty slice, the field is treated as wildcard.
//
// Command corresponds to a work of the row.
// Command[0] is used as command. Rest are treated as arguments for the command.
type Row struct {
	Minute  *Minutes  `json:"minute"`
	Hour    *Hours    `json:"hour"`
	Day     *Days     `json:"day"`
	Month   *Months   `json:"month"`
	Weekday *Weekdays `json:"weekday"`
	Command []string  `json:"command"`
}

func (r Row) IsWeekly() bool {
	return !r.Weekday.IsZero()
}

func (r Row) IsYearly() bool {
	return r.Weekday.IsZero() && !r.Month.IsZero()
}

func (r Row) IsMonthly() bool {
	return r.Weekday.IsZero() && r.Month.IsZero() && !r.Day.IsZero()
}

func (r Row) IsDaily() bool {
	return r.Weekday.IsZero() && r.Month.IsZero() && r.Day.IsZero() && !r.Hour.IsZero()
}

func (r Row) IsHourly() bool {
	return r.Weekday.IsZero() && r.Month.IsZero() && r.Day.IsZero() && r.Hour.IsZero() && !r.Minute.IsZero()
}

func (r Row) IsReboot() bool {
	return r.Weekday.IsZero() && r.Month.IsZero() && r.Day.IsZero() && r.Hour.IsZero() && r.Minute.IsZero()
}

func (r Row) IsValid() (ok bool, reason []string) {
	if !r.Minute.IsValid() {
		reason = append(reason, fmt.Sprintf("invalid minute: %v", *r.Minute))
	}
	if !r.Hour.IsValid() {
		reason = append(reason, fmt.Sprintf("invalid hour: %v", *r.Hour))
	}
	if !r.Day.IsValid() {
		reason = append(reason, fmt.Sprintf("invalid day: %v", *r.Day))
	}
	if !r.Month.IsValid() {
		reason = append(reason, fmt.Sprintf("invalid month: %v", *r.Month))
	}
	if !r.Weekday.IsValid() {
		reason = append(reason, fmt.Sprintf("invalid weekDay: %v", *r.Weekday))
	}
	if !r.IsCommandValid() {
		reason = append(reason, fmt.Sprintf("command is invalid"))
	}

	return len(reason) == 0, reason
}

func (r Row) IsCommandValid() bool {
	return r.Command != nil && len(r.Command) != 0
}

func (r Row) NextSchedule(now time.Time) (time.Time, error) {
	if ok, reason := r.IsValid(); !ok {
		return time.Time{}, fmt.Errorf("%v: %s", ErrMalformed, strings.Join(reason, ", "))
	}

	if r.IsReboot() {
		return now, nil
	}

	var next time.Time
	year := now.Year()
	weekdays := r.Weekday.Get()
	for i := 0; i <= 1; i++ {
	monthLoop:
		for _, m := range r.Month.Get() {
		dayLoop:
			for _, d := range r.Day.Get() {
			hourLoop:
				for _, h := range r.Hour.Get() {
				minLoop:
					for _, min := range r.Minute.Get() {
						next = time.Date(year+i, m, int(d), int(h), int(min), 0, 0, now.Location())
						if next.Month() != m {
							// Strong indication of overflow.
							// Maybe leap year.
							next = lastDayOfMonth(next.AddDate(0, -1, 0))
						}
						if next.After(now) && weekdayContain(next.Weekday(), weekdays) {
							return next, nil
						}
						sub := now.Sub(next)
						switch {
						case sub > 365*24*time.Hour:
							break monthLoop
						case sub > 31*24*time.Hour:
							break dayLoop
						case sub > 24*time.Hour:
							break hourLoop
						case sub > time.Hour:
							break minLoop
						}
					}
				}
			}
		}
	}
	return now, ErrNoMatch
}

func (r Row) GetCommand() []string {
	if r.IsCommandValid() {
		return r.Command
	} else {
		return nil
	}
}

func lastDayOfMonth(t time.Time) time.Time {
	year, month, _ := t.Date()
	h, m, s := t.Clock()
	nano := t.Nanosecond()
	return time.Date(
		year,
		month,
		1,
		h,
		m,
		s,
		nano,
		t.Location(),
	).AddDate(0, 1, -1)
}

func weekdayContain(target time.Weekday, sl []time.Weekday) bool {
	for _, w := range sl {
		if target == w {
			return true
		}
	}
	return false
}
