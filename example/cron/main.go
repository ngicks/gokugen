package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ngicks/gokugen"
	"github.com/ngicks/gokugen/cron"
	"github.com/ngicks/gokugen/scheduler"
)

func main() {
	if err := _main(); err != nil {
		panic(err)
	}
}

func printNowWithId(workId string) gokugen.WorkFnWParam {
	return func(ctxCancelCh, taskCancelCh <-chan struct{}, scheduled time.Time, param any) (any, error) {
		now := time.Now()
		var isCtxCancelled, isTaskCancelled bool
		select {
		case <-ctxCancelCh:
			isCtxCancelled = true
		default:
		}
		select {
		case <-taskCancelCh:
			isTaskCancelled = true
		default:
		}

		fmt.Printf(
			"workId: %s, scheduled: %s, diff to now: %s, isCtxCancelled: %t, isTaskCancelled: %t, param: %v\n",
			workId,
			scheduled.Format(time.RFC3339Nano),
			now.Sub(scheduled).String(),
			isCtxCancelled,
			isTaskCancelled,
			param,
		)

		return nil, nil
	}
}

//go:embed cron_tab.json
var tabBin []byte
var tab cron.CronTab

func init() {
	err := json.Unmarshal(tabBin, &tab)
	if err != nil {
		panic(err)
	}
}

func rowToController(
	tab []cron.RowLike,
	whence time.Time,
	scheduler cron.Scheduler,
	registry cron.WorkRegistry,
) (reschedulers []*cron.CronLikeRescheduler) {
	for _, v := range tab {
		res := cron.NewCronLikeRescheduler(
			v,
			whence,
			func(workErr error, callCount int) bool { return true },
			scheduler,
			registry,
		)
		reschedulers = append(reschedulers, res)
	}
	return
}

type PseudoRegistry struct{}

func (pr PseudoRegistry) Load(key string) (value gokugen.WorkFnWParam, ok bool) {
	return printNowWithId(key), true
}

func _main() (err error) {
	now := time.Now()

	innerScheduler := scheduler.NewScheduler(5, 0)
	sched := gokugen.NewScheduler(innerScheduler)

	rows := make([]cron.RowLike, 0)
	for _, r := range tab {
		rows = append(rows, r)
	}
	rows = append(rows, cron.Duration{Duration: 30 * time.Second, Command: []string{"every_30_seconds"}})

	reschedulers := rowToController(rows, now, sched, PseudoRegistry{})

	for _, v := range reschedulers {
		err = v.Schedule()
		if err != nil {
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go innerScheduler.Start(ctx)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalCh:
		fmt.Printf("received singal: %s\n", sig)
	}

	for _, v := range reschedulers {
		v.Cancel()
	}
	cancel()
	innerScheduler.End()

	return
}
