package repository

import (
	"github.com/ngicks/gokugen/repository/gormmodel"
	// "gorm.io/driver/sqlite" // CGO version
	"github.com/glebarez/sqlite" // This would increase binary size around 2 MiB. If you don't like it, implement your own core
	"gorm.io/gorm"
)

func NewGorm(db *gorm.DB, hookTimer HookTimer) (*Repository, error) {
	core := NewDefaultGormCore(db)

	return New(core, hookTimer), nil
}

func NewSqlite3(dbPath string, opts ...gorm.Option) (*Repository, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), opts...)
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&gormmodel.Task{}, &gormmodel.MetaKeyValue{})
	if err != nil {
		return nil, err
	}

	hookTimer := NewHookTimer()

	return NewGorm(db, hookTimer)
}
