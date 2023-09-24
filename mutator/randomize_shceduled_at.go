package mutator

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
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
		randomizeScheduledAt.Max, err = parseDur(max)
		if err != nil {
			return RandomizeScheduledAt{}, false, err
		}
	}
	if minOk {
		randomizeScheduledAt.Min, err = parseDur(min)
		if err != nil {
			return RandomizeScheduledAt{}, false, err
		}
	}
	return randomizeScheduledAt, true, nil
}

func parseDur(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	dur, err := time.ParseDuration(s)
	if err == nil {
		return dur, nil
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return time.Duration(i), nil
	}

	return 0, fmt.Errorf(
		"The value of %s and %s must be formatted as"+
			" time.ParseDuration can understand or"+
			" simple int representing time.Duration. but input is %s",
		LabelRandomizeScheduledAtMax, LabelRandomizeScheduledAtMin, s,
	)
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
