package mutator

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/und/option"
)

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
		ScheduledAt: option.Some(p.ScheduledAt.Value().Add(time.Duration(randVal))),
	})
}
