package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ngicks/gokugen/dispatcher"
	"github.com/ngicks/gokugen/repository"
	"github.com/ngicks/gokugen/scheduler"
	syncparam "github.com/ngicks/type-param-common/sync-param"
	"github.com/ngicks/type-param-common/util"
)

func main() {
	workRegistry := syncparam.Map[string, dispatcher.WorkFn]{}

	nowInRFC3339Nano := func() string {
		return time.Now().Format(time.RFC3339Nano)
	}
	workRegistry.Store("log-1", func(ctx context.Context, param []byte) error {
		fmt.Printf("log-1: <%s> %s\n", nowInRFC3339Nano(), string(param))
		return nil
	})
	workRegistry.Store("log-2", func(ctx context.Context, param []byte) error {
		fmt.Printf("log-2: <%s> %s\n", nowInRFC3339Nano(), string(param))
		return nil
	})

	workerPoolDispatcher := dispatcher.NewWorkerPoolDispatcher(&workRegistry)
	workerPoolDispatcher.WorkerPool.Add(10)
	defer workerPoolDispatcher.WorkerPool.Wait()
	defer workerPoolDispatcher.WorkerPool.Remove(100)

	heapRepo := repository.NewHeapRepository()

	inMemoryScheduler := scheduler.New(workerPoolDispatcher, heapRepo, scheduler.PassThroughHook{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go inMemoryScheduler.Run(ctx, true, true)

	now := time.Now()
	var task2, task3 scheduler.Task
	for idx, param := range []scheduler.TaskParam{
		{
			ScheduledAt: now.Add(time.Second),
			WorkId:      "log-1",
			Param:       []byte("foo"),
		},
		{
			ScheduledAt: now.Add(7 * time.Second),
			WorkId:      "log-1",
			Param:       []byte("baz"),
		},
		{
			ScheduledAt: now.Add(2 * time.Second),
			WorkId:      "log-2",
			Param:       []byte("bar"),
		},
		{
			ScheduledAt: now.Add(7 * time.Second),
			WorkId:      "log-1",
			Param:       []byte("baz"),
		},
	} {
		task, err := inMemoryScheduler.AddTask(param)
		if err != nil {
			panic(err)
		}
		fmt.Printf(
			"scheduled for id %s for %s\n",
			task.Id, task.ScheduledAt.Format(time.RFC3339Nano),
		)
		if idx+1 == 2 {
			task2 = task
		}
		if idx+1 == 3 {
			task3 = task
		}
	}
	<-time.After(time.Millisecond)

	err := inMemoryScheduler.Cancel(task2.Id)
	if err != nil {
		panic(err)
	}
	fmt.Printf("cancelled task %s\n", task2.Id)

	p := scheduler.TaskParam{ScheduledAt: now.Add(3 * time.Second)}
	err = inMemoryScheduler.Update(task3.Id, p)
	if err != nil {
		panic(err)
	}
	fmt.Printf("updated task %s to %s\n", task2.Id, util.Must(json.Marshal(p)))

	<-time.After(10 * time.Second)
}
