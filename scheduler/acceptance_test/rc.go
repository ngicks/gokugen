package acceptancetest

import (
	"github.com/ngicks/gokugen/scheduler"
)

// oneYearLater is far-future time.
var oneYearLater = TruncatedNow().AddDate(1, 0, 0)

type ParamAndFiller struct {
	Param scheduler.TaskParam
	// Filler fills fields which are non-zero in TaskParam with random valid value.
	Filler func(t scheduler.Task) scheduler.Task
}

func fuzzParamFilter() []ParamAndFiller {
	return []ParamAndFiller{
		{
			scheduler.TaskParam{
				ScheduledAt: oneYearLater,
			},
			fillScheduledAt,
		},
		{
			scheduler.TaskParam{
				WorkId: RandStrLen(16),
			},
			fillWorkId,
		},
		{
			scheduler.TaskParam{
				Param: RandByteLen(64),
			},
			fillParm,
		},
		{
			scheduler.TaskParam{
				Meta: map[string]string{RandStrLen(15): RandStrLen(15)},
			},
			fillMeta,
		},
	}
}

func fillScheduledAt(t scheduler.Task) scheduler.Task {
	t.ScheduledAt = oneYearLater
	return t
}

func fillWorkId(t scheduler.Task) scheduler.Task {
	t.WorkId = "nah"
	return t
}

var randomBytes = RandByteLen(128)

func fillParm(t scheduler.Task) scheduler.Task {
	t.Param = append([]byte{}, randomBytes...)
	return t
}

var randomMeta map[string]string

func init() {
	mm := make(map[string]string)
	for i := 0; i < int(RandByteLen(1)[0]); i++ {
		mm[string(RandByteLen(16))] = string(RandByteLen(16))
	}
	randomMeta = mm
}

func fillMeta(t scheduler.Task) scheduler.Task {
	t.Meta = randomMeta
	return t
}
