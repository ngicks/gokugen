package rescheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/tidwall/gjson"
)

type RescheduleMeta struct {
	Id    string `json:"id"`
	Param []byte `json:"param"`
}

// RescheduleRule is set of scheduling rule, Schedule.
// It can be directly unmarshalled from json map where keys are string,
// and values are one of below:
//   - string literal that represents cron tab
//   - json serialized CronSchedule,
//   - LimitedSchedule,
//   - or IntervalSchedule.
type RescheduleRule map[string]Schedule

func (r *RescheduleRule) UnmarshalJSON(data []byte) error {
	rawMap := map[string]json.RawMessage{}

	err := json.Unmarshal(data, &rawMap)
	if err != nil {
		return err
	}

	for k, v := range rawMap {
		sched, err := UnmarshalSchedule(v)
		if err != nil {
			return err
		}
		(*r)[k] = sched
	}

	return nil
}

type ScheduleSpecId string

const (
	Cron     ScheduleSpecId = "cron"
	Limited  ScheduleSpecId = "limited"
	Interval ScheduleSpecId = "interval"
)

type Schedule interface {
	// Next returns next schedule time.
	Next(param []byte) (next time.Time, nextParam []byte, err error)
	Initial(from time.Time) (param []byte)
}

func UnmarshalSchedule(data []byte) (Schedule, error) {
	if data[0] == '"' {
		row := string(bytes.Trim(data, "\""))
		sched, err := cron.ParseStandard(row)
		if err != nil {
			return nil, err
		}
		return &CronSchedule{Row: row, Spec: sched}, nil
	} else if gjson.Get(string(data), "spec").Exists() {
		var s CronSchedule
		err := json.Unmarshal(data, &s)
		if err != nil {
			return nil, err
		}
		return &s, nil
	} else if gjson.Get(string(data), "n").Exists() {
		var s LimitedSchedule
		err := json.Unmarshal(data, &s)
		if err != nil {
			return nil, err
		}
		return &s, nil
	} else if gjson.Get(string(data), "dur").Exists() {
		var s IntervalSchedule
		err := json.Unmarshal(data, &s)
		if err != nil {
			return nil, err
		}
		return &s, nil
	}

	return nil, fmt.Errorf("unknown: %s", string(data))
}

type CronSchedule struct {
	Row  string        `json:"row,omitempty"` // non empty only if unmarshalled from cron expression string.
	Spec cron.Schedule `json:"spec"`
}

func (s *CronSchedule) Initial(from time.Time) (param []byte) {
	bin, _ := json.Marshal(CronScheduleParam{from, from})
	return bin
}

func (s *CronSchedule) MarshalJSON() ([]byte, error) {
	if s.Row != "" {
		return json.Marshal(s.Row)
	}

	// avoiding infinite recursive marshalling.
	//
	// must be in sync with CronSchedule
	type cronSched struct {
		Spec cron.Schedule `json:"spec"`
	}
	return json.Marshal(cronSched{s.Spec})
}

func (s *CronSchedule) UnmarshalJSON(data []byte) error {
	// must be in sync with CronSchedule
	type cronSched struct {
		Spec json.RawMessage `json:"spec"`
	}

	var sched cronSched
	err := json.Unmarshal(data, &sched)
	if err != nil {
		return err
	}

	var row string
	var impl cron.Schedule
	// NOTE: watch every addition of schedule implementation of robfig/cron
	if sched.Spec[0] == '"' {
		row = string(bytes.Trim(sched.Spec, "\""))
		spec, err := cron.ParseStandard(row)
		if err != nil {
			return err
		}
		impl = spec
	} else if gjson.Get(string(sched.Spec), "Second").Exists() {
		var cronSched cron.SpecSchedule
		err := json.Unmarshal(sched.Spec, &cronSched)
		if err != nil {
			return err
		}
		impl = &cronSched
	} else if gjson.Get(string(sched.Spec), "Delay").Exists() {
		var cronSched cron.ConstantDelaySchedule
		err := json.Unmarshal(sched.Spec, &cronSched)
		if err != nil {
			return err
		}
		impl = &cronSched
	} else {
		return fmt.Errorf("unknown: %s", string(sched.Spec))
	}

	s.Row = row // non empty only if input json is cron expression string.
	s.Spec = impl
	return nil
}

func (s *CronSchedule) Next(param []byte) (next time.Time, nextParam []byte, err error) {
	var p CronScheduleParam
	err = json.Unmarshal(param, &p)
	if err != nil {
		return time.Time{}, nil, err
	}

	next = s.Spec.Next(p.Next)
	nextParam, err = json.Marshal(CronScheduleParam{Prev: p.Next, Next: next})
	if err != nil {
		return time.Time{}, nil, err
	}

	return next, nextParam, nil
}

type CronScheduleParam struct {
	Prev time.Time
	Next time.Time
}

type LimitedSchedule struct {
	Schedule Schedule `json:"schedule"`
	N        int64    `json:"n"`
}

func (s *LimitedSchedule) Initial(from time.Time) (param []byte) {
	rest := s.Schedule.Initial(from)
	bin, _ := json.Marshal(LimitedScheduleParam{s.N, string(rest)})
	return bin
}

func (s *LimitedSchedule) UnmarshalJSON(data []byte) error {
	// must be in sync with LimitedSchedule
	type lim struct {
		Schedule json.RawMessage `json:"schedule"`
		N        int64           `json:"n"`
	}

	var l lim

	err := json.Unmarshal(data, &l)
	if err != nil {
		return err
	}

	sched, err := UnmarshalSchedule(l.Schedule)
	if err != nil {
		return err
	}

	if _, ok := sched.(*LimitedSchedule); ok {
		return fmt.Errorf("nested limited schedule")
	}

	s.N = l.N
	s.Schedule = sched
	return nil
}

func (s LimitedSchedule) Next(param []byte) (next time.Time, nextParam []byte, err error) {
	var p LimitedScheduleParam
	err = json.Unmarshal(param, &p)
	if err != nil {
		return time.Time{}, nil, err
	}

	if p.N <= 0 {
		return time.Time{}, nil, &Done{}
	}
	next, paramInner, err := s.Schedule.Next([]byte(p.Rest))
	if err != nil {
		return time.Time{}, nil, err
	}

	nextParam, err = json.Marshal(LimitedScheduleParam{N: p.N - 1, Rest: string(paramInner)})
	if err != nil {
		return time.Time{}, nil, err
	}

	return next, nextParam, nil
}

type LimitedScheduleParam struct {
	N    int64
	Rest string
}

type IntervalSchedule struct {
	Dur time.Duration `json:"dur"`
}

func (s *IntervalSchedule) Initial(from time.Time) (param []byte) {
	bin, _ := json.Marshal(IntervalScheduleParam{from, from})
	return bin
}

func (s *IntervalSchedule) Next(param []byte) (next time.Time, nextParam []byte, err error) {
	var p IntervalScheduleParam
	err = json.Unmarshal(param, &p)
	if err != nil {
		return time.Time{}, nil, err
	}

	next = p.Next.Add(s.Dur)

	nextParam, err = json.Marshal(IntervalScheduleParam{
		Prev: p.Next,
		Next: next,
	})

	if err != nil {
		return time.Time{}, nil, err
	}

	return next, nextParam, nil
}

type IntervalScheduleParam = CronScheduleParam
