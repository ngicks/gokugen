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

func TestGormAcceptance(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "test-gokugen-gorm-sqlite3-*")
	if err != nil {
		panic(err)
	}

	t.Logf("%s", dir)
	// in-memory sqlite3 db seemingly takes much power. Some CI env become slow with it.
	sqliteFilename := filepath.Join(dir, "sqlite3.db")

	defer os.Remove(sqliteFilename)

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
