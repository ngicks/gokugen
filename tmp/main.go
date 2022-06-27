package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/scheduler"
	taskstorage "github.com/ngicks/gokugen/task_storage"
)

func main() {
	scheduler := gokugen.NewMiddlewareApplicator(scheduler.NewScheduler(5, 0))

	// Repository interface.
	// External data storage is manipulated through this interface.
	var repository taskstorage.RepositoryUpdater
	// When Sync-ing, this cb is used to determine task should be restored
	// and re-scheduled in internal scheduler.
	// (e.g. ignore tasks if they are too old and overdue.)
	var shouldRestore func(taskstorage.TaskInfo) bool
	// workRegistry is used to retrieve work function associated to WorkId.
	var workRegisry interface {
		Load(key string) (value taskstorage.WorkFnWParam, ok bool)
	}
	// Context wrapper applicator function used in Sync.
	// In Sync newly created ctx is used to reschedule.
	// So without this function context wrapper
	// that should be applied in upper user code is totally ignored.
	var syncCtxWrapper func(gokugen.SchedulerContext) gokugen.SchedulerContext

	taskStorage := taskstorage.NewSingleNodeTaskStorage(
		repository,
		shouldRestore,
		workRegisry,
		syncCtxWrapper,
	)

	// Correct usage is as middleware.
	scheduler.Use(taskStorage.Middleware(true)...)

	// Sync syncs itnernal state with external.
	// Normally TaskStorage does it reversely through middlewares,
	// mirroring internal state to external data storage.
	// But after rebooting system, or repository is changed externally,
	// Sync is needed to fetch back external data.
	rescheduled, schedulingErr, err := taskStorage.Sync(scheduler.Schedule)
	if err != nil {
		panic(err)
	}

	for taskId, taskController := range rescheduled {
		fmt.Printf(
			"id = %s, is scheduled for = %s\n",
			taskId,
			taskController.GetScheduledTime().Format(time.RFC3339Nano),
		)
	}
	for taskId, schedulingErr := range schedulingErr {
		fmt.Printf("id = %s, err = %s\n", taskId, schedulingErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go scheduler.Scheduler().Start(ctx)

	var scheduleTarget time.Time
	task, err := scheduler.Schedule(
		// To store correct data to external repository,
		// WorkId, Param is additionally needed.
		gokugen.BuildContext(
			scheduleTarget,
			nil,
			nil,
			gokugen.WithWorkIdOption("func1"),
			gokugen.WithParamOption([]string{"param", "param"}),
		),
	)
	if err != nil {
		panic(err)
	}

	// This is wrapped scheduler.TaskController.
	task.IsCancelled()

	// some time later...

	// cancel ctx and tear down scheduler.
	cancel()
	scheduler.Scheduler().End()
}
