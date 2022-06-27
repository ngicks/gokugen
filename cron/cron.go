package cron

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ngicks/type-param-common/slice"
)

var (
	ErrMalformed = errors.New("malformed cron row")
	ErrNoMatch   = errors.New("no match")
)

type WorkFn = func(targeted, current time.Time, repeat int) (any, error)

type CronTab []Row

var _ RowLike = Row{}

// Row is crontab row like data structure.
// Nil or empty time fields are treated as wildcard that matches every value of corresponding field.
// Row with all nil time fields is `Reboot`. NextSchedule of `Reboot` returns simply passed `now`.
//
// Command corresponds to a work of the row.
// Command[0] is used as command. Rest are treated as arguments for the command.
//
// Data range is as below
//
// | field   | allowed value                 |
// | ------- | ----------------------------- |
// | Minute  | 0-59                          |
// | Hour    | 0-23                          |
// | Day     | 1-31                          |
// | Month   | 1-12                          |
// | Weekday | 0-6                           |
// | Command | command name, param, param... |
//
// When any of time fields has value out of this range or command is invalid, IsValid() returns (false, non-empty-slice).
type Row struct {
	Minute  *Minutes  `json:"minute,omitempty"`
	Hour    *Hours    `json:"hour,omitempty"`
	Day     *Days     `json:"day,omitempty"`
	Month   *Months   `json:"month,omitempty"`
	Weekday *Weekdays `json:"weekday,omitempty"`
	Command []string  `json:"command"`
}

// IsReboot returns true only if all 5 time fields are nil.
func (r Row) IsReboot() bool {
	return r.Weekday == nil && r.Month == nil && r.Day == nil && r.Hour == nil && r.Minute == nil
}

// IsValid checks if r is valid. ok is false when any of time fields are out of allowed range
// or command is invalid. reason string slice is always non nil.
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

// IsCommandValid returns false if Command is nil or empty. returns true otherwise.
func (r Row) IsCommandValid() bool {
	return r.Command != nil && len(r.Command) != 0
}

// NextSchedule returns time matched to r's configuration and most recent but after now.
// Returned time has same location as one of now.
// error is non nil if r is invalid.
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
						if next.After(now) && slice.Has(weekdays, next.Weekday()) {
							return next, nil
						}
						sub := now.Sub(next)
						switch {
						case sub > 365*24*time.Hour:
							isLeapYear := IsLeapYear(next)
							if !isLeapYear {
								break monthLoop
							} else if isLeapYear && sub > 366*24*time.Hour {
								break monthLoop
							}
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
	// Could this happen?
	return now, ErrNoMatch
}

// GetCommand returns []string if Command is valid, nil otherwise.
// Behavior after mutating returned slice is undefined.
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

func IsLeapYear(t time.Time) bool {
	year := time.Date(t.Year(), time.December, 31, 0, 0, 0, 0, time.UTC)
	return year.YearDay() == 366
}
