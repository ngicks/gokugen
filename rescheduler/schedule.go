package rescheduler

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/tidwall/gjson"
)

// RescheduleRule is set of scheduling rule, Schedule.
// It can be directly unmarshalled from json map that keys are string,
// and values are one of string literal that represents cron tab,
// json serialized CronSchedule, LimitedSchedule, or IntervalSchedule.
type RescheduleRule map[string]Schedule

func (r *RescheduleRule) UnmarshalJSON(data []byte) error {
	rawMap := map[string]json.RawMessage{}

	err := json.Unmarshal(data, &rawMap)
	if err != nil {
		return err
	}

	for k, v := range rawMap {
		sched, err := UnmarshalScheduler(v)
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
}
type ScheduleType string

type SavedSchedule struct {
	Type     ScheduleType    `json:"type"`
	Schedule json.RawMessage `json:"schedule"`
}

func UnmarshalScheduler(data []byte) (Schedule, error) {
	if data[0] == '"' {
		sched, err := cron.ParseStandard(string(bytes.Trim(data, "\"")))
		if err != nil {
			return nil, err
		}
		return &CronSchedule{Spec: sched}, nil
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
	Spec cron.Schedule `json:"spec"`
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

	var impl cron.Schedule
	// NOTE: watch every addition of schedule implementation of robfig/cron
	if gjson.Get(string(sched.Spec), "Second").Exists() {
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

	s.Spec = impl
	return nil
}

func (s *CronSchedule) Next(param []byte) (next time.Time, nextParam []byte, err error) {
	var p CronScheduleParam
	err = p.UnmarshalBinary(param)
	if err != nil {
		return time.Time{}, nil, err
	}

	next = s.Spec.Next(p.Next)
	nextParam, err = CronScheduleParam{Prev: p.Next, Next: next}.MarshalBinary()
	if err != nil {
		return time.Time{}, nil, err
	}

	return next, nextParam, nil
}

type CronScheduleParam struct {
	Prev time.Time
	Next time.Time
}

func (p CronScheduleParam) MarshalBinary() (data []byte, err error) {
	return marshal2Times(p.Prev, p.Next)
}

func (p *CronScheduleParam) UnmarshalBinary(data []byte) error {
	var err error
	p.Prev, p.Next, err = unmarshal2Times(data)
	if err != nil {
		return err
	}
	return nil
}

func marshal2Times(t1, t2 time.Time) ([]byte, error) {
	buf := new(bytes.Buffer)
	times := [2]int64{
		t1.UnixMilli(),
		t2.UnixMilli(),
	}
	err := binary.Write(buf, binary.BigEndian, times[:])
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unmarshal2Times(data []byte) (time.Time, time.Time, error) {
	var n [2]int64
	buf := bytes.NewBuffer(data)
	err := binary.Read(buf, binary.BigEndian, n[:])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return time.UnixMilli(n[0]), time.UnixMilli(n[1]), nil
}

type LimitedSchedule struct {
	Schedule Schedule `json:"schedule"`
	N        int64    `json:"n"`
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

	sched, err := UnmarshalScheduler(l.Schedule)
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
	err = p.UnmarshalBinary(param)
	if err != nil {
		return time.Time{}, nil, err
	}

	if p.N <= 0 {
		return time.Time{}, nil, nil
	}
	next, paramInner, err := s.Schedule.Next(p.Rest)
	if err != nil {
		return time.Time{}, nil, err
	}

	nextParam, err = LimitedScheduleParam{N: p.N - 1, Rest: paramInner}.MarshalBinary()
	if err != nil {
		return time.Time{}, nil, err
	}

	return next, nextParam, nil
}

type LimitedScheduleParam struct {
	N    int64
	Rest []byte
}

func (p LimitedScheduleParam) MarshalBinary() (data []byte, err error) {
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, p.N)
	if err != nil {
		return nil, err
	}
	return append(buf.Bytes(), p.Rest...), nil
}

func (p *LimitedScheduleParam) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("wrong len = %d, want >= 8", len(data))
	}

	nBuf, rest := data[:8], data[8:]

	var n int64
	buf := bytes.NewBuffer(nBuf)
	err := binary.Read(buf, binary.BigEndian, &n)
	if err != nil {
		return err
	}

	p.N = n
	p.Rest = rest
	return nil
}

type IntervalSchedule struct {
	Dur time.Duration `json:"dur"`
}

func (s *IntervalSchedule) Next(param []byte) (next time.Time, nextParam []byte, err error) {
	var p IntervalScheduleParam
	err = p.UnmarshalBinary(param)
	if err != nil {
		return time.Time{}, nil, err
	}

	next = p.Next.Add(s.Dur)

	nextParam, err = IntervalScheduleParam{
		Prev: p.Next,
		Next: next,
	}.MarshalBinary()

	if err != nil {
		return time.Time{}, nil, err
	}

	return next, nextParam, nil
}

type IntervalScheduleParam = CronScheduleParam
