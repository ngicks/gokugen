package repository

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	acceptancetest "github.com/ngicks/gokugen/scheduler/acceptance_test"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const inMemoryDb = ":memory:"

func TestGormAcceptance(t *testing.T) {
	var sqliteFilename string

	if os.Getenv("GOKUGEN_SQLITE3_INMEMORY") != "" {
		sqliteFilename = inMemoryDb
	} else {
		dir, err := os.MkdirTemp(os.TempDir(), "test-gokugen-gorm-sqlite3-*")
		if err != nil {
			panic(err)
		}

		// in-memory sqlite3 db seemingly takes much power. Some CI env become slow with it.
		sqliteFilename = filepath.Join(dir, "sqlite3.db")
	}

	t.Logf("%s", sqliteFilename)

	defer func() {
		if sqliteFilename != inMemoryDb {
			os.Remove(sqliteFilename)
		}
	}()

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,       // Disable color
		},
	)

	t.Logf("%+v\n", newLogger)

	conf := &gorm.Config{
		// Logger: newLogger,
	}

	repo, err := NewSqlite3(sqliteFilename, conf)
	if err != nil {
		panic(err)
	}

	acceptancetest.TestRepository(t, repo)
}
