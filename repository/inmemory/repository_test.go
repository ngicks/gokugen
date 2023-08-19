package inmemory

import (
	"testing"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/def/acceptance/repository"
)

func TestRepository(t *testing.T) {
	repository.TestRepository(t, func() def.Repository {
		return NewInMemoryRepository()
	})
}
