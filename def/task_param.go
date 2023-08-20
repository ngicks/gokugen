package def

import (
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ngicks/und/option"
)

// TaskUpdateParam is set of parameters for users to update tasks.
type TaskUpdateParam struct {
	WorkId      option.Option[string]                   `json:"work_id"`
	Priority    option.Option[int]                      `json:"priority"`
	Param       option.Option[map[string]string]        `json:"param"`
	Meta        option.Option[map[string]string]        `json:"meta"`
	ScheduledAt option.Option[time.Time]                `json:"scheduled_at"`
	Deadline    option.Option[option.Option[time.Time]] `json:"deadline"`
}

func (p TaskUpdateParam) Update(u TaskUpdateParam) TaskUpdateParam {
	p.WorkId = u.WorkId.Or(p.WorkId)
	p.Param = u.Param.Or(p.Param)
	p.Priority = u.Priority.Or(p.Priority)
	p.ScheduledAt = u.ScheduledAt.Or(p.ScheduledAt)
	p.Deadline = u.Deadline.Or(p.Deadline)
	p.Meta = u.Meta.Or(p.Meta)
	return p.Clone()
}

func (p TaskUpdateParam) ToTask(id string, createdAt time.Time) Task {
	return Task{
		Id:        id,
		CreatedAt: NormalizeTime(createdAt),
		State:     TaskScheduled,
	}.Update(p)
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

func (t TaskUpdateParam) Normalize() TaskUpdateParam {
	t = t.Clone()
	t.ScheduledAt = t.ScheduledAt.Map(NormalizeTime)
	t.Deadline = normalizeOptionOptionTime(t.Deadline)
	return t
}

func normalizeOptionOptionTime(
	v option.Option[option.Option[time.Time]],
) option.Option[option.Option[time.Time]] {
	return v.Map(
		func(v option.Option[time.Time]) option.Option[time.Time] {
			return v.Map(NormalizeTime)
		},
	)
}

// TaskQueryParam is used to query task.
type TaskQueryParam struct {
	Id           option.Option[string]                     `json:"id"`
	WorkId       option.Option[string]                     `json:"work_id"`
	Priority     option.Option[int]                        `json:"priority"`
	State        option.Option[State]                      `json:"state"`
	Err          option.Option[string]                     `json:"err"`
	Param        option.Option[[]MapMatcher]               `json:"param"`
	Meta         option.Option[[]MapMatcher]               `json:"meta"`
	ScheduledAt  option.Option[TimeMatcher]                `json:"scheduled_at"`
	CreatedAt    option.Option[TimeMatcher]                `json:"created_at"`
	Deadline     option.Option[option.Option[TimeMatcher]] `json:"deadline"`
	CancelledAt  option.Option[option.Option[TimeMatcher]] `json:"cancelled_at"`
	DispatchedAt option.Option[option.Option[TimeMatcher]] `json:"dispatched_at"`
	DoneAt       option.Option[option.Option[TimeMatcher]] `json:"done_at"`
}

func (m TaskQueryParam) Match(t Task) bool {
	return matchComparable(t.Id, m.Id) &&
		matchComparable(t.WorkId, m.WorkId) &&
		matchComparable(t.Priority, m.Priority) &&
		matchComparable(t.State, m.State) &&
		matchComparable(t.Err, m.Err) &&
		matchMap(t.Param, m.Param) &&
		matchMap(t.Meta, m.Meta) &&
		matchTime(&t.ScheduledAt, m.ScheduledAt) &&
		matchTime(&t.CreatedAt, m.CreatedAt) &&
		matchOptTime(t.Deadline.Plain(), m.Deadline) &&
		matchOptTime(t.CancelledAt.Plain(), m.CancelledAt) &&
		matchOptTime(t.DispatchedAt.Plain(), m.DispatchedAt) &&
		matchOptTime(t.DoneAt.Plain(), m.DoneAt)
}

func matchComparable[T comparable](v T, m option.Option[T]) bool {
	if m.IsNone() {
		return true
	}
	return v == m.Value()
}

func matchMap(v map[string]string, m option.Option[[]MapMatcher]) bool {
	if m.IsNone() {
		return true
	}
	return MapMatchers(m.Value()).Match(v)
}

func matchTime(v *time.Time, m option.Option[TimeMatcher]) bool {
	if m.IsNone() {
		return true
	}
	return m.Value().Match(v)
}

func matchOptTime(v *time.Time, m option.Option[option.Option[TimeMatcher]]) bool {
	if m.IsNone() {
		return true
	}
	if m.Value().IsNone() {
		return v == nil
	}
	return m.Value().Value().Match(v)
}

func (m TaskQueryParam) Clone() TaskQueryParam {
	m.Param = m.Param.Map(slices.Clone)
	m.Meta = m.Meta.Map(slices.Clone)
	return m
}

func (m TaskQueryParam) Normalize() TaskQueryParam {
	m.ScheduledAt = m.ScheduledAt.Map(normalizeTimeMatcher)
	m.CreatedAt = m.CreatedAt.Map(normalizeTimeMatcher)
	m.CancelledAt = normalizeOptionTimeMatcher(m.CancelledAt)
	m.DispatchedAt = normalizeOptionTimeMatcher(m.DispatchedAt)
	m.DoneAt = normalizeOptionTimeMatcher(m.DoneAt)
	return m.Clone()
}

func normalizeTimeMatcher(m TimeMatcher) TimeMatcher {
	m.Value = NormalizeTime(m.Value)
	return m
}

func normalizeOptionTimeMatcher(
	m option.Option[option.Option[TimeMatcher]],
) option.Option[option.Option[TimeMatcher]] {
	return m.Map(
		func(v option.Option[TimeMatcher]) option.Option[TimeMatcher] {
			return v.Map(normalizeTimeMatcher)
		},
	)
}

type MapMatcher struct {
	Key       string
	Value     string
	MatchType mapMatchType
}

func (m MapMatcher) Match(mm map[string]string) bool {
	switch m.MatchType.Get() {
	case MapMatcherHasKey:
		_, ok := mm[m.Key]
		return ok
	case MapMatcherExact, MapMatcherForward, MapMatcherBackward, MapMatcherMiddle:
		v, ok := mm[m.Key]
		if !ok {
			return false
		}
		switch m.MatchType.Get() {
		case MapMatcherExact:
			return v == m.Value
		case MapMatcherForward:
			return strings.HasPrefix(v, m.Value)
		case MapMatcherBackward:
			return strings.HasSuffix(v, m.Value)
		case MapMatcherMiddle:
			return strings.Contains(v, m.Value)
		}
	}
	return false
}

type MapMatchers []MapMatcher

func (mm MapMatchers) Match(v map[string]string) bool {
	for _, m := range mm {
		if !m.Match(v) {
			return false
		}
	}
	return true
}

type mapMatchType string

const (
	MapMatcherDefault  mapMatchType = ""
	MapMatcherHasKey   mapMatchType = "HasKey"   // query for objects which have that key.
	MapMatcherExact    mapMatchType = "Exact"    // query for exact same value.
	MapMatcherForward  mapMatchType = "Forward"  // query for forward match.
	MapMatcherBackward mapMatchType = "Backward" // query for backwward match.
	MapMatcherMiddle   mapMatchType = "Middle"   // query for partial match.
)

func (t mapMatchType) String() string {
	return string(t)
}

// Get returns mapMatchType, falling back to MapMatcherExact if unknown variants.
func (t mapMatchType) Get() mapMatchType {
	switch t {
	case MapMatcherHasKey, MapMatcherExact, MapMatcherForward,
		MapMatcherBackward, MapMatcherMiddle:
		return t
	default:
		return MapMatcherExact
	}
}

type TimeMatcher struct {
	MatchType timeMatchType
	Value     time.Time
}

func (m TimeMatcher) Match(v *time.Time) bool {
	switch m.MatchType.Get() {
	case TimeMatcherNonNull:
		return v != nil
	case TimeMatcherEqual, TimeMatcherBefore, TimeMatcherBeforeEqual,
		TimeMatcherAfter, TimeMatcherAfterEqual:
		if v == nil {
			return false
		}
		switch m.MatchType.Get() {
		case TimeMatcherEqual:
			return m.Value.Equal(*v)
		case TimeMatcherBefore:
			return m.Value.Compare(*v) > 0
		case TimeMatcherBeforeEqual:
			return m.Value.Compare(*v) >= 0
		case TimeMatcherAfter:
			return m.Value.Compare(*v) < 0
		case TimeMatcherAfterEqual:
			return m.Value.Compare(*v) <= 0
		}
	}
	return true
}

type timeMatchType string

const (
	TimeMatcherDefault     timeMatchType = ""
	TimeMatcherNonNull     timeMatchType = "NonNull"
	TimeMatcherEqual       timeMatchType = "Equal"
	TimeMatcherBefore      timeMatchType = "Before"
	TimeMatcherBeforeEqual timeMatchType = "BeforeEqual"
	TimeMatcherAfter       timeMatchType = "After"
	TimeMatcherAfterEqual  timeMatchType = "AfterEqual"
)

// Get returns timeMatchType, falling back to TimeMatcherEqual if unknown variants.
func (t timeMatchType) Get() timeMatchType {
	switch t {
	case TimeMatcherNonNull, TimeMatcherEqual, TimeMatcherBefore,
		TimeMatcherBeforeEqual, TimeMatcherAfter, TimeMatcherAfterEqual:
		return t
	default:
		return TimeMatcherEqual
	}
}
