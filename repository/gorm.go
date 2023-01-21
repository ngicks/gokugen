package repository

import (
	"github.com/ngicks/gokugen/repository/gormtask"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewGorm(db *gorm.DB, hookTimer HookTimer) *Repository {
	return &Repository{
		Core:      NewDefaultGormCore(db),
		HookTimer: hookTimer,
	}
}

func NewSqlite3(dbPath string, opts ...gorm.Option) (*Repository, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), opts...)
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	err = db.AutoMigrate(&gormtask.GormTask{})
	if err != nil {
		return nil, err
	}

	core := NewDefaultGormCore(db)
	hookTimer, err := NewHookTimer(core)
	if err != nil {
		return nil, err
	}

	return &Repository{Core: core, HookTimer: hookTimer}, nil
}
