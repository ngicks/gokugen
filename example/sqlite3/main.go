package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ngicks/gokugen/repository"
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/type-param-common/util"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,       // Disable color
		},
	)

	fmt.Printf("%+v\n", newLogger)

	conf := &gorm.Config{
		// Logger: newLogger,
	}

	sqlite3Repo, err := repository.NewSqlite3("./db", conf)

	if err != nil {
		panic(err)
	}

	_, err = sqlite3Repo.GetNext()
	if err == nil {
		panic("should return error")
	}
	fmt.Printf("next error: %+v\n", err)

	tasks := make([]scheduler.Task, 0)
	now := time.Now()
	for _, param := range []scheduler.TaskParam{
		{
			ScheduledAt: now.Add(time.Second),
			WorkId:      "log-1",
			Param:       []byte("foo"),
		},
		{
			ScheduledAt: now.Add(time.Second),
			Priority:    util.Escape(50),
			WorkId:      "log-1",
			Param:       []byte("foofoo"),
		},
		{
			ScheduledAt: now.Add(7 * time.Second),
			WorkId:      "log-1",
			Param:       []byte("baz"),
		},
		{
			ScheduledAt: now.Add(7 * time.Second),
			WorkId:      "log-1",
			Param:       []byte("bazbaz"),
			Priority:    util.Escape(100),
		},
		{
			ScheduledAt: now.Add(2 * time.Second),
			WorkId:      "log-2",
			Param:       []byte("bar"),
		},
	} {
		task, err := sqlite3Repo.AddTask(param)
		if err != nil {
			panic(err)
		}

		tasks = append(tasks, task)

		fmt.Printf(
			"scheduled for id %s for %s\n",
			task.Id, task.ScheduledAt.Format(time.RFC3339Nano),
		)
	}

	next, err := sqlite3Repo.GetNext()
	if err != nil {
		panic(err)
	}
	fmt.Printf("next: %+v\n", next)

	updated, err := sqlite3Repo.Update(tasks[0].Id, scheduler.TaskParam{Param: []byte("hi hu mi"), Priority: util.Escape(25)})
	fmt.Printf("updated = %t, err = %+v\n", updated, err)

	printTask := func(id string) {
		task, err := sqlite3Repo.GetById(id)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", task)
	}

	printTask(tasks[0].Id)

	updated, err = sqlite3Repo.Update(tasks[0].Id, scheduler.TaskParam{Param: []byte("hi hu mi"), Priority: util.Escape(25)})
	fmt.Printf("updated = %t, err = %+v\n", updated, err)
	printTask(tasks[0].Id)

	updated, err = sqlite3Repo.Update(tasks[0].Id, scheduler.TaskParam{Param: []byte("hi hu mi"), Priority: util.Escape(50)})
	fmt.Printf("updated = %t, err = %+v\n", updated, err)
	printTask(tasks[0].Id)

	_, err = sqlite3Repo.Update(scheduler.NeverExistenceId, scheduler.TaskParam{Priority: util.Escape(25)})
	fmt.Printf("%+v\n", err)

	c, err := sqlite3Repo.Cancel(tasks[0].Id)
	fmt.Printf("cancelled = %t, err = %+v\n", c, err)
	t, _ := sqlite3Repo.GetById(tasks[0].Id)
	fmt.Printf("%+v\n", t)

	c, err = sqlite3Repo.Cancel(tasks[0].Id)
	fmt.Printf("cancelled = %t, err = %+v\n", c, err)

	updated, err = sqlite3Repo.Update(tasks[0].Id, scheduler.TaskParam{Param: []byte("nah")})
	fmt.Printf("updated = %t, err = %+v\n", updated, err)
}
