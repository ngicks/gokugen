# gokugen

go+刻限(kokugen)

Gokugen is middleware-applicable scheduler built on top of min-heap backed, limitting number of concurrently-processing taks and in-memory scheduler.

刻限(kokugen) is japanese word that means an appointed time, scheduled time, or due.

## Idea

The idea is based on [this article](https://qiita.com/kawasin73/items/7af6766c7898a656b1ee)(written in japanese).

`./scheduler` contains similar but modified implementation.

### Differences

- It removes cancelled tasks from min-heap at every one minute.
- It passes 2 channels to task that would be closed if scheduler is ended and the task is cancelled respectively.
- It has countermeasure for abnormally-returned work (i.e. calling runtime.Goexit or panicking). But not tested yet!
- Task cancellations are controlled by Cancel method of struct instance returned from Schedule.
- Cancellation of scheduler is controlled by context.Context.

## Architecture

simplified architecture

![simplified_architecture](./arch.drawio.svg)

## TODO

- [x] Reimplement funtionality
  - [x] in-memory shceduler
  - [x] single node task storage middleware
  - [x] cron-like interface
- [x] Implement multi node task storage middleware
- [x] Refactoring
- [ ] example package
- [ ] Add detailed careful test.

## Packages

### ./scheduler

See `./example/simple/main.go ` for exmpale usage.

Scheduler is in-memory scheduler.

It stores task to min-heap, a priority queue that prioritize least scheduled time.
`Feeder` sets timer to min scheduled time. When timer is emitted one dedicated goroutine gets all tasks whose scheduled time is past. And then sends tasks to worker via a channel.

Scheduler needs 2+n goroutines, 1 for dispatch loop 1 for canceller loop and n for workers.

Workers can be increased or decreased dynamically. If a removed worker is doing work, it will be removed after work is done.

The delay between scheduled time and actual work invocation time is typically under 30 milli secs. But it should depend on environments. So you should your own benchmark at your own setup.

### ./heap

Min-heap with added Exclude and Peek method.

### ./cron

See `./example/cron/main.go ` for exmpale usage.

Cron-tab-row-like struct and cron-like rescheduler.

A row will never be scheduled twice or more simultaneously. Single row must be rescheduled after corresponding work is done.

- Use cron.Row to schedule task at every `n` minutes, or ever `n` day of `m` month, or alike.
- Use cron.Builder to build cron.Row easily.
- Use cron.Duration to simply schedule task at every specific interval.

### ./task_storage

See `./example/persistent_shceduler/main.go` for example usage.

TaskStorage make scheduled tasks persistent.

TaskStorage is middleware and sync-controller.

New(Single|Multi)TaskStorage.Middleware() returns middleware and those are pluggable part of gokugen.Scheduler.
These middlewares stores relevant information to persistent data storage by using struct implementing Repository, which is passed to New(Single|Multi)TaskStorage.

See `impl/repository` for example reposoitry implementations (in-memory for tests. sqlite3 for simple usage.)

Call `Sync()` after reboot of system or at interval. It will synchronize internal state with external repository. Scheduled tasks will be restored and continued. External cancel or whatever will remove correspoding task and infromation. Externally added tasks also will be scheduled internally.

Passing `true` to Middleware() enables param free feature. It let those middlewares to forget param until needed. This will surely adds overhead to tasks, but will reduce memory comsumption. If your workloads are expected to be heavy and needs to handle large number of tasks, this option might be helpful, especially when param is super big.
