package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/impl/repository"
	"github.com/ngicks/gokugen/scheduler"
	taskstorage "github.com/ngicks/gokugen/task_storage"
	syncparam "github.com/ngicks/type-param-common/sync-param"
)

func main() {
	if err := _main(); err != nil {
		panic(err)
	}
}

func printJsonIndent(j any) {
	b, err := json.MarshalIndent(j, "", "    ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}

func printNowWithId(workId string) gokugen.WorkFnWParam {
	return func(taskCtx context.Context, scheduled time.Time, param any) (any, error) {
		now := time.Now()
		var isCtxCancelled bool
		select {
		case <-taskCtx.Done():
			isCtxCancelled = true
		default:
		}

		fmt.Printf(
			"workId: %s, scheduled: %s, diff to now: %s, isCtxCancelled: %t, param: %v\n",
			workId,
			scheduled.Format(time.RFC3339Nano),
			now.Sub(scheduled).String(),
			isCtxCancelled,
			param,
		)

		return nil, nil
	}
}

func prepare(dbFilename string) (
	sched *gokugen.MiddlewareApplicator[*scheduler.Scheduler],
	repo *repository.Sqlite3Repo,
	workRegistory *syncparam.Map[string, gokugen.WorkFnWParam],
	taskStorage *taskstorage.SingleNodeTaskStorage,
	err error,
) {
	repo, err = repository.NewSql3Repo(dbFilename)
	if err != nil {
		return
	}
	workRegistory = &syncparam.Map[string, gokugen.WorkFnWParam]{}
	workRegistory.Store("func1", printNowWithId("func1"))
	workRegistory.Store("func2", printNowWithId("func2"))
	workRegistory.Store("func3", printNowWithId("func3"))

	sched = gokugen.NewMiddlewareApplicator(scheduler.NewScheduler(5, 0))

	taskStorage = taskstorage.NewSingleNodeTaskStorage(
		repo,
		func(ti taskstorage.TaskInfo) bool { return true },
		workRegistory,
		func(sc gokugen.SchedulerContext) gokugen.SchedulerContext {
			param, _ := gokugen.GetParam(sc)
			return gokugen.WrapContext(
				sc,
				gokugen.WithParam(
					map[string]any{
						"synced":   "yayyay",
						"paramOld": param,
					},
				),
			)
		},
	)
	sched.Use(taskStorage.Middleware(true)...)

	return
}

func _main() (err error) {
	p, err := os.MkdirTemp("", "sqlite3-tmp-*")
	if err != nil {
		return
	}
	dbFilename := filepath.Join(p, "db")

	sched, repo, _, _, err := prepare(dbFilename)
	if err != nil {
		return err
	}

	now := time.Now()

	sched.Schedule(
		gokugen.BuildContext(
			now,
			nil,
			nil,
			gokugen.WithWorkId("func1"),
			gokugen.WithParam([]string{"param", "param"}),
		),
	)
	sched.Schedule(
		gokugen.BuildContext(
			now,
			nil,
			nil,
			gokugen.WithWorkId("func2"),
			gokugen.WithParam([]string{"param", "param"}),
		),
	)
	sched.Schedule(
		gokugen.BuildContext(
			now,
			nil,
			nil,
			gokugen.WithWorkId("func3"),
			gokugen.WithParam([]string{"param", "param"}),
		),
	)

	ctx, cancel := context.WithDeadline(context.Background(), now.Add(2*time.Second))
	sched.Scheduler().Start(ctx)
	cancel()
	sched.Scheduler().End()

	fmt.Println("after 1st teardown: tasks in repository")
	taskInfos, err := repo.GetAll()
	if err != nil {
		return
	}
	for _, v := range taskInfos {
		printJsonIndent(v)
	}

	sched, _, _, taskStorage, err := prepare(dbFilename)
	if err != nil {
		return
	}

	rescheduled, schedulingErr, err := taskStorage.Sync(sched.Schedule)
	if err != nil {
		return
	}
	fmt.Println("restoed from persistent data storage:")
	printJsonIndent(rescheduled)
	printJsonIndent(schedulingErr)

	ctx, cancel = context.WithDeadline(context.Background(), now.Add(7*time.Second))
	sched.Scheduler().Start(ctx)
	cancel()
	sched.Scheduler().End()

	fmt.Println("after 2nd teardown: tasks in repository")
	taskInfos, err = repo.GetAll()
	if err != nil {
		return
	}
	for _, v := range taskInfos {
		printJsonIndent(v)
	}

	return
}
