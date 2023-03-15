package repository

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ngicks/gokugen/scheduler"
	acceptancetest "github.com/ngicks/gokugen/scheduler/acceptance_test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	inMemoryDb       = "file::memory:"
	sharedInMemoryDb = "file::memory:?cache=shared"
)

func getDbFilename() string {
	if e := os.Getenv("GOKUGEN_SQLITE3_INMEMORY"); e != "" && e != "0" {
		return inMemoryDb
	} else {
		dir, err := os.MkdirTemp(os.TempDir(), "test-gokugen-gorm-sqlite3-*")
		if err != nil {
			panic(err)
		}

		// in-memory sqlite3 db seemingly causes some CI env slow.
		// Default is an ordinary disk file in temp dir.
		return filepath.Join(dir, "sqlite3.db")
	}
}

func TestGormAcceptance(t *testing.T) {
	var (
		lock      sync.Mutex
		filenames []string
	)

	factory := func() scheduler.TaskRepository {
		sqliteFilename := getDbFilename()

		newLogger := logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Info, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,       // Disable color
			},
		)

		conf := &gorm.Config{
			Logger: newLogger,
		}

		repo, err := NewSqlite3(sqliteFilename, conf)
		if err != nil {
			panic(err)
		}

		t.Log(sqliteFilename)

		lock.Lock()
		filenames = append(filenames, sqliteFilename)
		lock.Unlock()

		return repo
	}

	defer func() {
		lock.Lock()
		defer lock.Unlock()
		for _, filename := range filenames {
			if filename != inMemoryDb {
				os.Remove(filename)
				os.Remove(filepath.Dir(filename))
			}
		}
	}()

	acceptancetest.TestRepository(t, factory, acceptancetest.RepositoryTestConfig{
		FindMetaContain: acceptancetest.FindMetaContainTestConfig{
			// Forward:  true,
			// Backward: true,
			// Partial:  true,
		},
		DeleteBefore:     true,
		RevertDispatched: true,
	})
}

func TestGormReconnect(t *testing.T) {
	require := require.New(t)

	sqliteFilename := getDbFilename()

	if sqliteFilename == inMemoryDb {
		sqliteFilename = sharedInMemoryDb
	}

	t.Logf("%s", sqliteFilename)

	defer func() {
		if sqliteFilename != sharedInMemoryDb {
			os.Remove(sqliteFilename)
		}
	}()

	repo, err := NewSqlite3(sqliteFilename)
	if err != nil {
		panic(err)
	}

	tasks, err := addRandomTask(repo, 100)
	require.NoError(err)

	reconnected, err := NewSqlite3(sqliteFilename)
	require.NoError(err)

	for _, task := range tasks {
		got, err := reconnected.GetById(task.Id)
		require.NoError(err)

		if diff := cmp.Diff(task, got); diff != "" {
			t.Fatalf("not equal: %s", diff)
		}
	}
}
