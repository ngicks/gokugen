package repository

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	taskstorage "github.com/ngicks/gokugen/task_storage"
)

var _ taskstorage.RepositoryUpdater = &Sqlite3Repo{}

const tableSqlite string = `
CREATE TABLE IF NOT EXISTS taskInfo(
    id TEXT NOT NULL PRIMARY KEY,
    work_id TEXT NOT NULL,
    param BLOB,
    scheduled_time INTEGER NOT NULL,
    state TEXT NOT NULL,
    inserted_at INTEGER NOT NULL DEFAULT (strftime('%s','now') || substr(strftime('%f','now'),4)),
    last_modified_at INTEGER NOT NULL DEFAULT (strftime('%s','now') || substr(strftime('%f','now'),4))
) STRICT;`

const triggerUpdateSqlite string = `
CREATE TRIGGER IF NOT EXISTS trigger_update_taskInfo_last_modified_at AFTER UPDATE ON taskInfo
BEGIN
    UPDATE taskInfo SET last_modified_at = strftime('%s','now') || substr(strftime('%f','now'),4) WHERE id == NEW.id;
END;`

const insertSqlite string = `
INSERT INTO taskInfo(
	id,
	work_id,
	param,
	scheduled_time,
	state
) VALUES(?,?,?,?,?)
`

type sqlite3TaskInfo struct {
	Id               string
	Work_id          string
	Param            []byte
	Scheduled_time   int64
	State            string
	Inserted_at      int64
	Last_modified_at int64
}

func fromHighLevel(
	taskId string,
	taskInfo taskstorage.TaskInfo,
) (lowTaskInfo sqlite3TaskInfo, err error) {
	if taskInfo.State < 0 {
		err = taskstorage.ErrInvalidEnt
		return
	}
	// Implementation detail: param must be json marshalable
	var paramMarshaled []byte
	if taskInfo.Param != nil {
		paramMarshaled, err = json.Marshal(taskInfo.Param)
		if err != nil {
			return
		}
	}

	return sqlite3TaskInfo{
		Id:             taskId,
		Work_id:        taskInfo.WorkId,
		Param:          paramMarshaled,
		Scheduled_time: taskInfo.ScheduledTime.UnixMilli(),
		State:          taskInfo.State.String(),
	}, nil
}

func (ti sqlite3TaskInfo) toHighLevel() (taskInfo taskstorage.TaskInfo, err error) {
	var param any
	if ti.Param != nil && len(ti.Param) != 0 {
		err = json.Unmarshal(ti.Param, &param)
		if err != nil {
			return
		}
	}
	state := taskstorage.NewStateFromString(ti.State)
	if state < 0 {
		err = taskstorage.ErrInvalidEnt
		return
	}
	taskInfo = taskstorage.TaskInfo{
		Id:            ti.Id,
		WorkId:        ti.Work_id,
		Param:         param,
		ScheduledTime: time.UnixMilli(ti.Scheduled_time),
		State:         state,
		LastModified:  time.UnixMilli(ti.Last_modified_at),
	}
	return
}

type Sqlite3Repo struct {
	randomStr *RandStringGenerator
	mu        sync.RWMutex
	db        *sql.DB
}

func NewSql3Repo(dbName string) (repo *Sqlite3Repo, err error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return
	}

	for _, stmt := range []string{tableSqlite, triggerUpdateSqlite} {
		if err := execStmt(db, stmt); err != nil {
			return nil, err
		}
	}

	return &Sqlite3Repo{
		randomStr: NewRandStringGenerator(time.Now().UnixMicro(), 16, hex.NewEncoder),
		db:        db,
	}, nil
}

func createTable(db *sql.DB) (err error) {
	stmt, err := db.Prepare(tableSqlite)
	if err != nil {
		return
	}
	_, err = stmt.Exec()
	if err != nil {
		return
	}
	return
}

func execStmt(db *sql.DB, stmtQuery string) (err error) {
	stmt, err := db.Prepare(stmtQuery)
	if err != nil {
		return
	}
	_, err = stmt.Exec()
	if err != nil {
		return
	}
	return
}

func (r *Sqlite3Repo) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.db.Close()
}

func (r *Sqlite3Repo) Insert(taskInfo taskstorage.TaskInfo) (taskId string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	taskId, err = r.randomStr.Generate()
	if err != nil {
		return
	}

	lowlevel, err := fromHighLevel(taskId, taskInfo)
	if err != nil {
		return
	}

	stmt, err := r.db.Prepare(insertSqlite)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		lowlevel.Id,
		lowlevel.Work_id,
		lowlevel.Param,
		lowlevel.Scheduled_time,
		lowlevel.State,
	)

	if err != nil {
		return
	}

	return
}

func (r *Sqlite3Repo) fetchAllForQuery(query string, exec ...any) (taskInfos []taskstorage.TaskInfo, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stmt, err := r.db.Prepare(query)
	if err != nil {
		return
	}
	rows, err := stmt.Query(exec...)
	if err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		lowlevel := sqlite3TaskInfo{}
		err = rows.Scan(
			&lowlevel.Id,
			&lowlevel.Work_id,
			&lowlevel.Param,
			&lowlevel.Scheduled_time,
			&lowlevel.State,
			&lowlevel.Inserted_at, // this is unused. TODO: change `select *`` to like `select all but inserted_at` query.
			&lowlevel.Last_modified_at,
		)
		if err != nil {
			return
		}
		var taskInfo taskstorage.TaskInfo
		taskInfo, err = lowlevel.toHighLevel()
		if err != nil {
			return
		}
		taskInfos = append(taskInfos, taskInfo)
	}
	return
}

func (r *Sqlite3Repo) GetAll() (taskInfos []taskstorage.TaskInfo, err error) {
	return r.fetchAllForQuery("SELECT * FROM taskInfo ORDER BY last_modified_at ASC")
}

func (r *Sqlite3Repo) GetUpdatedSince(since time.Time) ([]taskstorage.TaskInfo, error) {
	return r.fetchAllForQuery("SELECT * FROM taskInfo WHERE last_modified_at >= ? ORDER BY last_modified_at ASC", since.UnixMilli())
}

func (r *Sqlite3Repo) GetById(taskId string) (taskInfo taskstorage.TaskInfo, err error) {
	infos, err := r.fetchAllForQuery("SELECT * FROM taskInfo WHERE id = ?", taskId)
	if err != nil {
		return
	}
	if len(infos) == 0 {
		err = taskstorage.ErrNoEnt
		return
	}
	return infos[0], nil
}

func (r *Sqlite3Repo) markState(id string, new taskstorage.TaskState) (ok bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stmt, err := r.db.Prepare("UPDATE taskInfo SET state = ? WHERE id = ? AND (state = ? OR state = ?)")
	if err != nil {
		return
	}
	res, err := stmt.Exec(new.String(), id, taskstorage.Initialized.String(), taskstorage.Working.String())
	if err != nil {
		return
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return
	}
	if affected <= 0 {
		return false, nil
	}
	return true, nil
}

func (r *Sqlite3Repo) MarkAsDone(id string) (ok bool, err error) {
	return r.markState(id, taskstorage.Done)
}
func (r *Sqlite3Repo) MarkAsCancelled(id string) (ok bool, err error) {
	return r.markState(id, taskstorage.Cancelled)
}
func (r *Sqlite3Repo) MarkAsFailed(id string) (ok bool, err error) {
	return r.markState(id, taskstorage.Failed)
}

func (r *Sqlite3Repo) UpdateState(id string, old, new taskstorage.TaskState) (swapped bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	stmt, err := r.db.Prepare("UPDATE taskInfo SET state = ? WHERE id = ? AND state = ?")
	if err != nil {
		return
	}
	res, err := stmt.Exec(new.String(), id, old.String())
	if err != nil {
		return
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return
	}
	if affected <= 0 {
		return false, nil
	}
	return true, nil
}

func (r *Sqlite3Repo) Update(id string, diff taskstorage.UpdateDiff) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	args := make([]any, 0)

	setter := make([]string, 0)
	lowlevel, err := fromHighLevel(id, diff.Diff)
	if err != nil {
		return
	}
	if diff.UpdateKey.WorkId {
		setter = append(setter, "work_id = ?")
		args = append(args, lowlevel.Work_id)
	}
	if diff.UpdateKey.Param {
		setter = append(setter, "param = ?")
		args = append(args, lowlevel.Param)
	}
	if diff.UpdateKey.ScheduledTime {
		setter = append(setter, "scheduled_time = ?")
		args = append(args, lowlevel.Scheduled_time)
	}
	if diff.UpdateKey.State {
		setter = append(setter, "state = ?")
		args = append(args, lowlevel.State)
	}

	if len(args) == 0 {
		// no-op
		return
	}

	query := "UPDATE taskInfo SET " + strings.Join(setter, ", ") + " WHERE id = ? AND state = ?"
	args = append(args, []any{id, taskstorage.Initialized.String()}...)

	stmt, err := r.db.Prepare(query)
	if err != nil {
		return
	}
	res, err := stmt.Exec(args...)
	if err != nil {
		return
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return
	}
	if affected <= 0 {
		return taskstorage.ErrNoEnt
	}
	return

}
