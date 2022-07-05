package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/impl/repository"
	workregistry "github.com/ngicks/gokugen/impl/work_registry"
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

type Param1 struct {
	Foo string
	Bar string
}

func (p Param1) Mix() string {
	var ret string
	for range p.Foo {
		for range p.Bar {
			ret += string(p.Foo[rand.Int63n(int64(len(p.Foo)))])
			ret += string(p.Bar[rand.Int63n(int64(len(p.Bar)))])
		}
	}
	return ret
}

type Param2 struct {
	Baz string
	Qux string
}

func (p Param2) Double() string {
	var runes []rune
	for _, bRune := range p.Baz {
		for _, qRune := range p.Qux {
			runes = append(runes, bRune)
			runes = append(runes, qRune)
		}
	}
	return string(runes)
}

func printNowWithIdParam1(workId string) gokugen.WorkFnWParam {
	return func(taskCtx context.Context, scheduled time.Time, param any) (any, error) {
		printer(workId, taskCtx, scheduled, param.(Param1).Mix())
		return nil, nil
	}
}

func printNowWithIdParam2(workId string) gokugen.WorkFnWParam {
	return func(taskCtx context.Context, scheduled time.Time, param any) (any, error) {
		printer(workId, taskCtx, scheduled, param.(Param2).Double())
		return nil, nil
	}
}

func printer(workId string, taskCtx context.Context, scheduled time.Time, param string) {
	now := time.Now()
	var isCtxCancelled bool
	select {
	case <-taskCtx.Done():
		isCtxCancelled = true
	default:
	}

	fmt.Printf(
		"workId: %s, scheduled: %s, diff to now: %s, isCtxCancelled: %t, param: %s\n",
		workId,
		scheduled.Format(time.RFC3339Nano),
		now.Sub(scheduled).String(),
		isCtxCancelled,
		param,
	)
}

func prepare(dbFilename string) (
	sched *gokugen.MiddlewareApplicator[*scheduler.Scheduler],
	repo *repository.Sqlite3Repo,
	workRegistory taskstorage.WorkRegistry,
	taskStorage *taskstorage.SingleNodeTaskStorage,
	err error,
) {
	repo, err = repository.NewSql3Repo(dbFilename)
	if err != nil {
		return
	}
	innerRegistry := &syncparam.Map[string, gokugen.WorkFnWParam]{}
	innerRegistry.Store("func1", printNowWithIdParam1("func1"))
	innerRegistry.Store("func2", printNowWithIdParam2("func2"))
	innerRegistry.Store("func3", printNowWithIdParam2("func3"))

	unmarshallerResgitry := &workregistry.TransformerRegistryImpl{}
	workregistry.AddType[Param1]("func1", unmarshallerResgitry)
	workregistry.AddType[Param2]("func2", unmarshallerResgitry)
	workregistry.AddType[Param2]("func3", unmarshallerResgitry)

	workRegistory = workregistry.NewParamTransformer(innerRegistry, unmarshallerResgitry)

	sched = gokugen.NewMiddlewareApplicator(scheduler.NewScheduler(5, 0))

	taskStorage = taskstorage.NewSingleNodeTaskStorage(
		repo,
		func(ti taskstorage.TaskInfo) bool { return true },
		workRegistory,
		nil,
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
			gokugen.WithParam(Param1{Foo: "param", Bar: "qqq"}),
		),
	)
	sched.Schedule(
		gokugen.BuildContext(
			now.Add(1*time.Second),
			nil,
			nil,
			gokugen.WithWorkId("func2"),
			gokugen.WithParam(Param2{Baz: "baaz", Qux: "param"}),
		),
	)
	sched.Schedule(
		gokugen.BuildContext(
			now.Add(3*time.Second),
			nil,
			nil,
			gokugen.WithWorkId("func3"),
			gokugen.WithParam(Param2{Baz: "quuux", Qux: "coorrge"}),
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
	for k, v := range rescheduled {
		fmt.Println(k, v.GetScheduledTime())
	}

	ctx, cancel = context.WithDeadline(context.Background(), now.Add(5*time.Second))
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
