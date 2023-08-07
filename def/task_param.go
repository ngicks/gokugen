package def

import (
	"time"

	"github.com/ngicks/gokugen/scheduler/util"
	"github.com/ngicks/und/option"
	"golang.org/x/exp/maps"
)

type TaskParam struct {
	WorkId       option.Option[string]                   `json:"work_id"`
	Param        option.Option[map[string]string]        `json:"param"`
	Priority     option.Option[int]                      `json:"priority"`
	State        option.Option[State]                    `json:"state"`
	ScheduledAt  option.Option[time.Time]                `json:"scheduled_at"`
	CreatedAt    option.Option[time.Time]                `json:"created_at"`
	CancelledAt  option.Option[option.Option[time.Time]] `json:"cancelled_at"`
	DispatchedAt option.Option[option.Option[time.Time]] `json:"dispatched_at"`
	DoneAt       option.Option[option.Option[time.Time]] `json:"done_at"`
	Err          option.Option[string]                   `json:"err"`
	Meta         option.Option[map[string]string]        `json:"meta"`
}

func (p TaskParam) ToTask(ignoreMicro bool) Task {
	return Task{}.Update(p, ignoreMicro)
}

func (p TaskParam) Clone() TaskParam {
	p.Param = p.Param.Map(
		func(v map[string]string) map[string]string {
			return maps.Clone(v)
		},
	)
	p.Meta = p.Meta.Map(
		func(v map[string]string) map[string]string {
			return maps.Clone(v)
		},
	)
	return p
}

func (t TaskParam) TruncTime() TaskParam {
	t = t.Clone()

	t.ScheduledAt = t.ScheduledAt.Map(util.DropMicros)
	t.CreatedAt = t.CreatedAt.Map(util.DropMicros)
	t.CancelledAt = t.CancelledAt.Map(
		func(v option.Option[time.Time]) option.Option[time.Time] {
			return v.Map(util.DropMicros)
		},
	)
	t.DispatchedAt = t.DispatchedAt.Map(
		func(v option.Option[time.Time]) option.Option[time.Time] {
			return v.Map(util.DropMicros)
		},
	)
	t.DoneAt = t.DoneAt.Map(
		func(v option.Option[time.Time]) option.Option[time.Time] {
			return v.Map(util.DropMicros)
		},
	)

	return t
}

type SearchMatcher struct {
	TaskParam
	Param option.Option[[]KeyValuePairMatcher] `json:"param"`
	Meta  option.Option[[]KeyValuePairMatcher] `json:"meta"`
}
