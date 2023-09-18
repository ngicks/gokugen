package mutator

import (
	"context"

	"github.com/ngicks/gokugen/def"
)

const (
	LabelRandomizeScheduledAtMin = "ngicks.RandomizeScheduledAt.min"
	LabelRandomizeScheduledAtMax = "ngicks.RandomizeScheduledAt.max"
	LabelScheduleAtNow           = "ngicks.ScheduleAtNow"
)

type ParamMutatingRepository struct {
	def.ObservableRepository
	MutatorStore MutatorStore
}

func (r *ParamMutatingRepository) AddTask(
	ctx context.Context,
	param def.TaskUpdateParam,
) (def.Task, error) {
	mutator, err := r.MutatorStore.Load(param.Meta.Value())
	if err != nil {
		return def.Task{}, err
	}
	param = mutator.Apply(param)
	return r.ObservableRepository.AddTask(ctx, param)
}

var DefaultMutatorStore MutatorStore = defaultMutatorStore{}

type MutatorStore interface {
	Load(meta map[string]string) (Mutators, error)
}

type defaultMutatorStore struct{}

type decoderFn func(meta map[string]string) (Mutator, bool, error)

var decoders = []decoderFn{
	func(meta map[string]string) (Mutator, bool, error) { return DecodeScheduleAtNow(meta) },
	func(meta map[string]string) (Mutator, bool, error) { return DecodeRandomizeScheduledAt(meta) },
}

func (defaultMutatorStore) Load(meta map[string]string) (Mutators, error) {
	if len(meta) == 0 {
		return make(Mutators, 0), nil
	}

	mutators := make(Mutators, 0)
	for _, dec := range decoders {
		mutator, found, err := dec(meta)
		if err != nil {
			return nil, err
		}
		if found {
			mutators = append(mutators, mutator)
		}
	}
	return mutators, nil
}
