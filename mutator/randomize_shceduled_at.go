package mutator

import (
	"math/rand"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixMicro()))
)

type Mutator interface {
	Mutate(p def.TaskUpdateParam) def.TaskUpdateParam
}

type Mutators []Mutator

func (m Mutators) Apply(param def.TaskUpdateParam) def.TaskUpdateParam {
	for _, mm := range m {
		param = mm.Mutate(param)
	}
	return param.Clone()
}

var _ Mutator = RandomizeScheduledAt{}

type RandomizeScheduledAt struct {
	Min, Max time.Duration
}

func (r RandomizeScheduledAt) Mutate(p def.TaskUpdateParam) def.TaskUpdateParam {
	diff := r.Max - r.Min
	positive := true
	if diff < 0 {
		positive = false
		diff = -diff
	}
	randVal := random.Int63n(int64(diff))
	if !positive {
		randVal = -randVal
	}
	return p.Update(def.TaskUpdateParam{
		ScheduledAt: option.Some(p.ScheduledAt.Value().Add(time.Duration(randVal))),
	})
}
