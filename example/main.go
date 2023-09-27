package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/ngicks/genericsync"
	"github.com/ngicks/gokugen/cron"
	"github.com/ngicks/gokugen/def"
	"github.com/ngicks/gokugen/dispatcher/workerpool"
	"github.com/ngicks/gokugen/repository"
	"github.com/ngicks/gokugen/repository/inmemory"
	"github.com/ngicks/gokugen/scheduler"
	"github.com/ngicks/und/option"
)

func workFoo(ctx context.Context, param map[string]string) error {
	bin, _ := json.Marshal(param)
	log.Printf("foo. param = %s\n", bin)
	return nil
}

func workBar(ctx context.Context, param map[string]string) error {
	bin, _ := json.Marshal(param)
	log.Printf("bar. param = %s\n", bin)
	return nil
}

func workBaz(ctx context.Context, param map[string]string) error {
	bin, _ := json.Marshal(param)
	log.Printf("baz. param = %s\n", bin)
	return nil
}

const rowsJson = `
[
	{
		"Schedule": "@every 7s",
		"Param": {
			"work_id": "foo",
    		"param": {
				"mah": ""
			}
		}
	}
]
`

func main() {
	rows := cron.Rows{}
	if err := rows.UnmarshalJSON([]byte(rowsJson)); err != nil {
		panic(err)
	}
	log.Printf("rows = %+#v\n", rows)

	registry := genericsync.Map[string, *def.WorkFn]{}
	foo := workFoo
	bar := workBar
	baz := workBaz
	registry.Store("foo", &foo)
	registry.Store("bar", &bar)
	registry.Store("baz", &baz)

	workerPoolDispatcher := workerpool.NewWorkerPoolDispatcher(&registry)

	workerPoolDispatcher.WorkerPool.Add(10)
	workerPoolDispatcher.WorkerPool.WaitUntil(func(alive, sleeping, active int) bool {
		return alive == 10
	})
	defer workerPoolDispatcher.WorkerPool.Wait()
	defer workerPoolDispatcher.WorkerPool.Remove(100)

	memRepo := inmemory.NewInMemoryRepository()
	now := time.Now()
	_, _ = memRepo.AddTask(context.Background(), def.TaskUpdateParam{
		WorkId:      option.Some("foo"),
		Param:       option.Some(map[string]string{"memRepo": ""}),
		ScheduledAt: option.Some(now.Add(5 * time.Second)),
	})
	_, _ = memRepo.AddTask(context.Background(), def.TaskUpdateParam{
		WorkId:      option.Some("bar"),
		Param:       option.Some(map[string]string{"memRepo": ""}),
		ScheduledAt: option.Some(now.Add(15 * time.Second)),
	})
	_, _ = memRepo.AddTask(context.Background(), def.TaskUpdateParam{
		WorkId:      option.Some("baz"),
		Param:       option.Some(map[string]string{"memRepo": ""}),
		ScheduledAt: option.Some(now.Add(25 * time.Second)),
	})

	repo := repository.New(memRepo, repository.NewMutationHookTimer())

	cronTasks, err := cron.NewCronTable(rows.Entries(time.Now()))
	if err != nil {
		panic(err)
	}

	repo.StartTimer(context.Background())
	cronTasks.StartTimer(context.Background())

	repoScheduler := scheduler.NewScheduler(repo, workerPoolDispatcher)
	cronScheduler := scheduler.NewVolatileTask(cronTasks, workerPoolDispatcher)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = repoScheduler.RunQueue(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = cronScheduler.RunQueue(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			state := repoScheduler.Step(ctx)
			log.Printf("repo state = %s\n", state.State())
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			state := cronScheduler.Step(ctx)
			log.Printf("cron state = %s\n", state.State())
		}
	}()
}
