package mutator

import (
	"crypto/rand"

	"github.com/ngicks/gokugen/def"
)

var (
	randomReader = rand.Reader
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
