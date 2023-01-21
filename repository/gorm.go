package repository

import (
	"github.com/ngicks/gokugen/repository/gormtask"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewGorm(db *gorm.DB, hookTimer HookTimer) (*Repository, error) {
	core := NewDefaultGormCore(db)

	err := core.db.AutoMigrate(&gormtask.GormTask{})
	if err != nil {
		return nil, err
	}

	return New(core, hookTimer), nil
}

func NewSqlite3(dbPath string, opts ...gorm.Option) (*Repository, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), opts...)
	if err != nil {
		return nil, err
	}

	hookTimer := NewHookTimer()

	return NewGorm(db, hookTimer)
}
