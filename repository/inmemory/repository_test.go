package inmemory

import (
	"os"
	"testing"

	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/def/acceptance/repository"
)

var (
	debug = os.Getenv("TEST_DEBUG") == "1"
)

func TestRepository_plain_go(t *testing.T) {
	repository.TestRepository(t, func() def.Repository {
		return NewInMemoryRepository()
	},
		debug,
	)
}
