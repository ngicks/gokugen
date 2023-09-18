package mutator

import (
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/mockable"
	"github.com/ngicks/und/option"
)

func DecodeScheduleAtNow(meta map[string]string) (ScheduleAtNow, bool, error) {
	_, ok := meta[LabelScheduleAtNow]
	if !ok {
		return ScheduleAtNow{}, false, nil
	}
	return ScheduleAtNow{}, true, nil
}

var _ Mutator = ScheduleAtNow{}

var clock mockable.Clock = mockable.NewClockReal()

type ScheduleAtNow struct {
}

func (s ScheduleAtNow) Mutate(p def.TaskUpdateParam) def.TaskUpdateParam {
	p.ScheduledAt = option.Some(clock.Now())
	p.Normalize()
	return p
}
