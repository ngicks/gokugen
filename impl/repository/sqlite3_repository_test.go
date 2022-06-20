package repository_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ngicks/gokugen/impl/repository"
	taskstorage "github.com/ngicks/gokugen/task_storage"
)

func newDbFilename() string {
	p, err := os.MkdirTemp("", "sqlite3-tmp-*")
	if err != nil {
		panic(err)
	}
	return filepath.Join(p, "db")
}

func TestSqlite3Repo(t *testing.T) {
	dbFilename := newDbFilename()

	fmt.Println(dbFilename)

	db, err := repository.NewSql3Repo(dbFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.UnixMilli(time.Now().UnixMilli())
	taskId, err := db.Insert(taskstorage.TaskInfo{
		WorkId:        "foobarbaz",
		Param:         nil,
		ScheduledTime: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	taskInfo, err := db.GetById(taskId)

	if err != nil {
		t.Fatal(err)
	}
	if taskInfo.Id != taskId {
		t.Fatalf("%s != %s", taskId, taskInfo.Id)
	} else if taskInfo.Param != nil {
		t.Fatalf("%v != %v", nil, taskInfo.Param)
	} else if taskInfo.ScheduledTime != now {
		t.Fatalf("%s != %s", now.Format(time.RFC3339Nano), taskInfo.ScheduledTime.Format(time.RFC3339Nano))
	}

	_, err = db.Insert(taskstorage.TaskInfo{
		WorkId:        "qux",
		Param:         map[string]string{"foo": "bar"},
		ScheduledTime: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Insert(taskstorage.TaskInfo{
		WorkId:        "corge",
		Param:         []string{"foo", "bar"},
		ScheduledTime: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	taskInfos, err := db.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	var lastUpdated time.Time
	for _, v := range taskInfos {
		if lastUpdated.Before(v.LastModified) {
			lastUpdated = v.LastModified
		}
	}

	time.Sleep(2 * time.Millisecond)

	err = db.Update(taskId, taskstorage.UpdateDiff{
		UpdateKey: taskstorage.UpdateKey{
			WorkId:        true,
			Param:         true,
			ScheduledTime: true,
			State:         true,
		},
		Diff: taskstorage.TaskInfo{
			WorkId:        "bazbaz",
			Param:         true,
			ScheduledTime: lastUpdated,
			State:         taskstorage.Done,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	taskInfos, err = db.GetUpdatedSince(lastUpdated.Add(time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if len(taskInfos) != 1 {
		t.Fatalf("must be 1 but %d", len(taskInfos))
	}
	taskInfo = taskInfos[0]

	if taskInfo.Id != taskId {
		t.Fatalf("%s != %s", taskId, taskInfo.Id)
	} else if taskInfo.Param != true {
		t.Fatalf("%v != %v", true, taskInfo.Param)
	} else if taskInfo.ScheduledTime != lastUpdated {
		t.Fatalf("%s != %s", lastUpdated.Format(time.RFC3339Nano), taskInfo.ScheduledTime.Format(time.RFC3339Nano))
	}
}

func TestSqlite3RepoSearchResultOrder(t *testing.T) {
	var err error

	db, err := repository.NewSql3Repo(newDbFilename())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Insert(taskstorage.TaskInfo{
		WorkId:        "0",
		Param:         nil,
		ScheduledTime: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Microsecond)
	_, err = db.Insert(taskstorage.TaskInfo{
		WorkId:        "1",
		Param:         nil,
		ScheduledTime: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Microsecond)
	_, err = db.Insert(taskstorage.TaskInfo{
		WorkId:        "2",
		Param:         nil,
		ScheduledTime: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}

	all, err := db.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if !(all[0].WorkId == "0" && all[1].WorkId == "1" && all[2].WorkId == "2") {
		t.Fatalf("wrong query: %v", all)
	}

	updatedSince, err := db.GetUpdatedSince(all[1].LastModified)
	if err != nil {
		t.Fatal(err)
	}
	if !(updatedSince[0].WorkId == "1" && updatedSince[1].WorkId == "2") {
		t.Fatalf("wrong query: %v", updatedSince)
	}
}
