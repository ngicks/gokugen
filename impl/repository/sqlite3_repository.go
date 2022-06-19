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

	// Implementation detail: param must be json marshalable
	var paramMarshaled []byte
	if taskInfo.Param != nil {
		paramMarshaled, err = json.Marshal(taskInfo.Param)
		if err != nil {
			return
		}
	}

	taskId, err = r.randomStr.Generate()
	if err != nil {
		return
	}

	stmt, err := r.db.Prepare(insertSqlite)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		taskId,
		taskInfo.WorkId,
		paramMarshaled,
		taskInfo.ScheduledTime.UnixMilli(),
		taskInfo.State.String(),
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
		var taskId, workId, stateRaw string
		var paramMarshaled []byte
		var scheduledTime, insertedAt, lastModifiedAt int64
		err = rows.Scan(
			&taskId,
			&workId,
			&paramMarshaled,
			&scheduledTime,
			&stateRaw,
			&insertedAt, // this is unused. TODO: change `select *`` to like `select all but inserted_at` query.
			&lastModifiedAt,
		)
		if err != nil {
			return
		}
		var param any
		if paramMarshaled != nil && len(paramMarshaled) != 0 {
			err = json.Unmarshal(paramMarshaled, &param)
			if err != nil {
				return
			}
		}
		state := taskstorage.NewStateFromString(stateRaw)
		if state < 0 {
			return nil, taskstorage.ErrInvalidEnt
		}
		ent := taskstorage.TaskInfo{
			Id:            taskId,
			WorkId:        workId,
			Param:         param,
			ScheduledTime: time.UnixMilli(scheduledTime),
			State:         state,
			LastModified:  time.UnixMilli(lastModifiedAt),
		}
		taskInfos = append(taskInfos, ent)
	}
	return
}

func (r *Sqlite3Repo) GetAll() (taskInfos []taskstorage.TaskInfo, err error) {
	return r.fetchAllForQuery("SELECT * FROM taskInfo")
}

func (r *Sqlite3Repo) GetUpdatedAfter(since time.Time) ([]taskstorage.TaskInfo, error) {
	return r.fetchAllForQuery("SELECT * FROM taskInfo WHERE last_modified_at >= ?", since.UnixMilli())
}

func (r *Sqlite3Repo) GetById(taskId string) (taskInfo taskstorage.TaskInfo, err error) {
	infos, err := r.fetchAllForQuery("SELECT * FROM taskInfo WHERE id=?", taskId)
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
	if diff.UpdateKey.WorkId {
		setter = append(setter, "work_id = ?")
		args = append(args, diff.Diff.WorkId)
	}
	if diff.UpdateKey.Param {
		setter = append(setter, "param = ?")
		var paramMarshaled []byte
		if diff.Diff.Param != nil {
			paramMarshaled, err = json.Marshal(diff.Diff.Param)
			if err != nil {
				return
			}
		}
		args = append(args, paramMarshaled)
	}
	if diff.UpdateKey.ScheduledTime {
		setter = append(setter, "scheduled_time = ?")
		args = append(args, diff.Diff.ScheduledTime.UnixMilli())
	}
	if diff.UpdateKey.State {
		setter = append(setter, "state = ?")
		args = append(args, diff.Diff.State.String())
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
