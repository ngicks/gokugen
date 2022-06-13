package cron

import "time"

// Minute is 0 - 59 uint value.
// Value out of this range is invalid and IsValid returns false.
type Minute uint

func (m Minute) IsValid() bool {
	if m > 59 {
		return false
	}
	return true
}

func (m Minute) Round() Minute {
	if m > 59 {
		return 59
	}
	return m
}

// Hour is 0 - 23 uint value.
// Value out of this range is invalid and IsValid returns false.
type Hour uint

func (h Hour) IsValid() bool {
	if h > 23 {
		return false
	}
	return true
}

func (h Hour) Round() Hour {
	if h > 23 {
		return 23
	}
	return h
}

// Day is 1 - 31 uint value.
// Value out of this range is invalid and IsValid returns false.
type Day uint

func (d Day) IsValid() bool {
	if d == 0 {
		return false
	} else if d > 31 {
		return false
	}
	return true
}

func (d Day) Round() Day {
	if d == 0 {
		return 1
	} else if d > 31 {
		return 31
	}
	return d
}

type Minutes []Minute

func (m *Minutes) IsZero() bool {
	return m == nil || *m == nil || len(*m) == 0
}

func (m *Minutes) Get() []Minute {
	if m.IsZero() {
		return m.Default()
	}
	sl := make([]Minute, 0)
	for _, v := range m.Default() {
		for _, min := range *m {
			if min == v {
				sl = append(sl, v)
				break
			}
		}
	}
	return sl
}

func (m *Minutes) Default() []Minute {
	minutes := make([]Minute, 0)
	for i := 0; i < 60; i++ {
		minutes = append(minutes, Minute(i))
	}
	return minutes
}

func (m *Minutes) IsValid() bool {
	// nil = wildcard(*).
	if m == nil || *m == nil {
		return true
	}
	for _, min := range *m {
		if !min.IsValid() {
			return false
		}
	}
	return true
}

type Hours []Hour

func (h *Hours) IsZero() bool {
	return h == nil || *h == nil || len(*h) == 0
}

func (h *Hours) Get() []Hour {
	if h.IsZero() {
		return h.Default()
	}
	sl := make([]Hour, 0)
	for _, v := range h.Default() {
		for _, hour := range *h {
			if hour == v {
				sl = append(sl, v)
				break
			}
		}
	}
	return sl
}

func (h *Hours) Default() []Hour {
	hours := make([]Hour, 0)
	for i := 0; i < 24; i++ {
		hours = append(hours, Hour(i))
	}
	return hours
}

func (h *Hours) IsValid() bool {
	// nil = wildcard(*).
	if h == nil || *h == nil {
		return true
	}
	for _, hour := range *h {
		if !hour.IsValid() {
			return false
		}
	}
	return true
}

type Days []Day

func (d *Days) IsZero() bool {
	return d == nil || *d == nil || len(*d) == 0
}
func (d *Days) Get() []Day {
	if d.IsZero() {
		return d.Default()
	}
	sl := make([]Day, 0)
	for _, v := range d.Default() {
		for _, dd := range *d {
			if dd == v {
				sl = append(sl, v)
				break
			}
		}
	}
	return sl
}

func (d *Days) Default() []Day {
	days := make([]Day, 0)
	for i := 1; i <= 31; i++ {
		days = append(days, Day(i))
	}
	return days
}

func (d *Days) IsValid() bool {
	// nil = wildcard(*).
	if d == nil || *d == nil {
		return true
	}
	for _, day := range *d {
		if !day.IsValid() {
			return false
		}
	}
	return true
}

func isValidMonth(m time.Month) bool {
	return time.January <= m && m <= time.December
}

func isValidWeekday(w time.Weekday) bool {
	return w == time.Sunday ||
		w == time.Monday ||
		w == time.Tuesday ||
		w == time.Wednesday ||
		w == time.Thursday ||
		w == time.Friday ||
		w == time.Saturday
}

type Months []time.Month

func (m *Months) IsZero() bool {
	return m == nil || *m == nil || len(*m) == 0
}

func (m *Months) Get() []time.Month {
	if m.IsZero() {
		return m.Default()
	}
	sl := make([]time.Month, 0)
	for _, v := range m.Default() {
		for _, mon := range *m {
			if mon == v {
				sl = append(sl, v)
				break
			}
		}
	}
	return sl
}

func (m *Months) Default() []time.Month {
	return []time.Month{
		time.January,
		time.February,
		time.March,
		time.April,
		time.May,
		time.June,
		time.July,
		time.August,
		time.September,
		time.October,
		time.November,
		time.December,
	}
}

func (m *Months) IsValid() bool {
	// nil = wildcard(*).
	if m == nil || *m == nil {
		return true
	}
	for _, month := range *m {
		if !isValidMonth(month) {
			return false
		}
	}
	return true
}

type Weekdays []time.Weekday

func (w *Weekdays) IsZero() bool {
	return w == nil || *w == nil || len(*w) == 0
}

func (w *Weekdays) Get() []time.Weekday {
	if w.IsZero() {
		return w.Default()
	}

	sl := make([]time.Weekday, 0)
	for _, v := range w.Default() {
		for _, wd := range *w {
			if wd == v {
				sl = append(sl, v)
				break
			}
		}
	}
	return sl
}

func (w *Weekdays) Default() []time.Weekday {
	return []time.Weekday{
		time.Sunday,
		time.Monday,
		time.Tuesday,
		time.Wednesday,
		time.Thursday,
		time.Friday,
		time.Saturday,
	}
}

func (w *Weekdays) IsValid() bool {
	// nil = wildcard(*).
	if w == nil || *w == nil {
		return true
	}
	for _, wd := range *w {
		if !isValidWeekday(wd) {
			return false
		}
	}
	return true
}
