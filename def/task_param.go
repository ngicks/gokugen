package def

import (
	"maps"
	"slices"
	"time"

	"github.com/ngicks/gokugen/def/util"
	"github.com/ngicks/und/option"
)

// TaskUpdateParam is set of parameters for users to update tasks.
type TaskUpdateParam struct {
	WorkId      option.Option[string]                   `json:"work_id"`
	Param       option.Option[map[string]string]        `json:"param"`
	Priority    option.Option[int]                      `json:"priority"`
	ScheduledAt option.Option[time.Time]                `json:"scheduled_at"`
	Deadline    option.Option[option.Option[time.Time]] `json:"deadline"`
	Meta        option.Option[map[string]string]        `json:"meta"`
}

func (p TaskUpdateParam) Update(u TaskUpdateParam) TaskUpdateParam {
	p = p.Clone()
	p.WorkId = u.WorkId.Or(p.WorkId)
	p.Param = u.Param.Or(p.Param)
	p.Priority = u.Priority.Or(p.Priority)
	p.ScheduledAt = u.ScheduledAt.Or(p.ScheduledAt)
	p.Deadline = u.Deadline.Or(p.Deadline)
	p.Meta = u.Meta.Or(p.Meta)
	return p
}

func (p TaskUpdateParam) ToTask(ignoreMicro bool) Task {
	return Task{}.Update(p, ignoreMicro)
}

func (p TaskUpdateParam) Clone() TaskUpdateParam {
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

func (t TaskUpdateParam) TruncTime() TaskUpdateParam {
	t = t.Clone()
	t.ScheduledAt = t.ScheduledAt.Map(util.DropMicros)
	t.Deadline = truncOptionOptionTime(t.Deadline)
	return t
}

func truncOptionOptionTime(
	v option.Option[option.Option[time.Time]],
) option.Option[option.Option[time.Time]] {
	return v.Map(
		func(v option.Option[time.Time]) option.Option[time.Time] {
			return v.Map(util.DropMicros)
		},
	)
}

// TaskSerachMatcher is used to query task.
type TaskQueryParam struct {
	Id           option.Option[string]                     `json:"id"`
	WorkId       option.Option[string]                     `json:"work_id"`
	Param        option.Option[[]MapMatcher]               `json:"param"`
	Meta         option.Option[[]MapMatcher]               `json:"meta"`
	Priority     option.Option[int]                        `json:"priority"`
	Err          option.Option[string]                     `json:"err"`
	State        option.Option[State]                      `json:"state"`
	ScheduledAt  option.Option[TimeMatcher]                `json:"scheduled_at"`
	CreatedAt    option.Option[TimeMatcher]                `json:"created_at"`
	Deadline     option.Option[option.Option[TimeMatcher]] `json:"deadline"`
	CancelledAt  option.Option[option.Option[TimeMatcher]] `json:"cancelled_at"`
	DispatchedAt option.Option[option.Option[TimeMatcher]] `json:"dispatched_at"`
	DoneAt       option.Option[option.Option[TimeMatcher]] `json:"done_at"`
}

func (m TaskQueryParam) Clone() TaskQueryParam {
	m.Param = m.Param.Map(slices.Clone)
	m.Meta = m.Meta.Map(slices.Clone)
	return m
}

func (m TaskQueryParam) TruncTime() TaskQueryParam {
	m = m.Clone()
	m.ScheduledAt = m.ScheduledAt.Map(truncTimeMatcher)
	m.CreatedAt = m.CreatedAt.Map(truncTimeMatcher)
	m.CancelledAt = truncOptionTimeMatcher(m.CancelledAt)
	m.DispatchedAt = truncOptionTimeMatcher(m.DispatchedAt)
	m.DoneAt = truncOptionTimeMatcher(m.DoneAt)
	return m
}

func truncTimeMatcher(m TimeMatcher) TimeMatcher {
	m.Value = util.DropMicros(m.Value)
	return m
}

func truncOptionTimeMatcher(
	m option.Option[option.Option[TimeMatcher]],
) option.Option[option.Option[TimeMatcher]] {
	return m.Map(
		func(v option.Option[TimeMatcher]) option.Option[TimeMatcher] {
			return v.Map(truncTimeMatcher)
		},
	)
}

type MapMatcher struct {
	Key       string
	Value     string
	MatchType mapMatchType
}

type mapMatchType string

const (
	MapMatcherHasKey   mapMatchType = "HasKey"   // query for objects which have that key.
	MapMatcherExact    mapMatchType = "Exact"    // query for exact same value.
	MapMatcherForward  mapMatchType = "Forward"  // query for forward match.
	MapMatcherBackward mapMatchType = "Backward" // query for backwward match.
	MapMatcherMiddle   mapMatchType = "Middle"   // query for partial match.
)

func (t mapMatchType) String() string {
	return string(t)
}

type TimeMatcher struct {
	MatchType timeMatchType
	Value     time.Time
}

type timeMatchType string

const (
	TimeMatcherNonNull     timeMatchType = "NonNull"
	TimeMatcherEqual       timeMatchType = "Equal"
	TimeMatcherBefore      timeMatchType = "Before"
	TimeMatcherBeforeEqual timeMatchType = "BeforeEqual"
	TimeMatcherAfter       timeMatchType = "After"
	TimeMatcherAfterEqual  timeMatchType = "AfterEqual"
)
