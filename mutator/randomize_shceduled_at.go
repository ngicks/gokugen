package mutator

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
)

func DecodeRandomizeScheduledAt(meta map[string]string) (RandomizeScheduledAt, bool, error) {
	max, maxOk := meta[LabelRandomizeScheduledAtMax]
	min, minOk := meta[LabelRandomizeScheduledAtMin]
	if !maxOk && !minOk {
		return RandomizeScheduledAt{}, false, nil
	}
	var (
		randomizeScheduledAt RandomizeScheduledAt
		err                  error
	)
	if maxOk {
		randomizeScheduledAt.Max, err = time.ParseDuration(max)
		if err != nil {
			return RandomizeScheduledAt{}, false, err
		}
	}
	if minOk {
		randomizeScheduledAt.Min, err = time.ParseDuration(min)
		if err != nil {
			return RandomizeScheduledAt{}, false, err
		}
	}
	return randomizeScheduledAt, true, nil
}

var _ Mutator = RandomizeScheduledAt{}

type RandomizeScheduledAt struct {
	Min, Max time.Duration
}

func (r RandomizeScheduledAt) Mutate(p def.TaskUpdateParam) def.TaskUpdateParam {
	diff := r.Max - r.Min
	neg := false
	if diff < 0 {
		neg = true
		diff = -diff
	}

	randBigVal, err := rand.Int(randomReader, big.NewInt(int64(diff)))
	if err != nil {
		panic(err)
	}

	randVal := randBigVal.Int64()
	if neg {
		randVal = -randVal
	}

	return p.Update(def.TaskUpdateParam{
		ScheduledAt: option.Some(p.ScheduledAt.Value().Add(r.Min + time.Duration(randVal))),
	})
}
