package mutator

import (
	"encoding/json"
	"fmt"
)

const (
	IdKey    = "github.com/ngicks/gokugen/mutator.id"
	ParamKey = "github.com/ngicks/gokugen/mutator.param"
)

var DefaultMutatorStore MutatorStore = defaultMutatorStore{}

type MutatorStore interface {
	Load(meta map[string]string) (Mutators, error)
}

type defaultMutatorStore struct{}

var decoders = map[string]func(param string) (Mutator, error){
	"RandomizeScheduledAt": func(param string) (Mutator, error) {
		var v RandomizeScheduledAt
		err := json.Unmarshal([]byte(param), &v)
		if err != nil {
			return nil, err
		}
		return v, nil
	},
}

func (defaultMutatorStore) Load(meta map[string]string) (Mutators, error) {
	mutators := make(Mutators, 0)
	for k, dec := range decoders {
		param := meta[k]
		mut, err := dec(param)
		if err != nil {
			return nil, fmt.Errorf("%w: wrong param for %s. input meta = %+#v", err, k, meta)
		}
		mutators = append(mutators, mut)
	}
	return mutators, nil
}
