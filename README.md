# gokugen [![Godoc](https://godoc.org/github.com/ngicks/gokugen?status.svg)](https://godoc.org/github.com/ngicks/gokugen)

go+刻限(kokugen)

The term 刻限(kokugen) is a japanese word equivalent of appointed time, scheduled time, or due.

gokugen is set of interfaces and a scheduler implementation. Mainly focused on single-process scheduler.

## Idea

The idea is based on [this article](https://qiita.com/kawasin73/items/7af6766c7898a656b1ee)(written in japanese).

This repository breaks up implementation into pieces, adds slight improvements, and recomposes them.

## The original story

I was thinking what was the best idea to get used with the concurrent programming in Go.
Back then, I had to make some job-scheduling system for my job, where the task needed to be scheduled some after it was added, and it needed to be cancelled if someone had sent the mail.

I decided to write my own job-scheduling system to gain some Go experiences.

In my job, things run in on-premise, relatively less-powerful machine. So everything should be in a single process, and runs at small memory footprint, and traffic is expected to be small. So main focus of this project is that: single process, small dependency, yet useful.

## Arch

![arch.drawio.svg](./arch.drawio.svg)

Things are decomposed to 3 elements.

- Scheduler
  - main loop of dispatching, task updates, and state-observations.
  - `Task` is intentionally defined to be serializable. Sending tasks over network is possible. But not planned.
- TaskRepository
  - Store and optionally save tasks to disk.
  - Watch every Repository mutation and set timer to the next scheduled task.
- Dispatcher
  - bridge to workers (executor).
  - block the main scheduler loop until at least a worker is available.

## Implementations

### Scheduler

- `Scheduler` runs a loop in which it waits for the next scheduled Task stored in `Repository`, and when the next scheduled time passed, it dispatches Tasks to workers through `Dispatcher`.
- It also runs a loop that serializes updates to the tasks.
  - It aims to avoid race conditions in updates, and allow last-second update block task dispatching.
- Scheduler also provide a way to observe its state changes and events via `LoopHooks`.
  - In `LoopHooks`, you may suppress non critical Repository specific errors, wait for `Repository` coming back from error state if it has temporality been down, or even retry Task update.

### TaskRepository

- `TaskRepository` is a combination of `RepositoryLike` interface and `TimerLike` interface.
  - It provides the typical Repository interface, and
  - It provides special search capability; GetNext() returns the next scheduled Task.
  - Also it watches task addition, completion and cancellation. It resets the timer to the next scheduled time if that is changed.
- Currently the project has 2 implementation in ./repository.
  - `HeapRepository`: in-memory Repository
    - It is backed by `map` and `container/heap`. The complexity of Task addition and removal is `O(log n)`.
  - `GormRepository`: persistent Repository using the ORM; `gorm.io/gorm`.
    - you can do everything by your self, or use default implementations: `GormCore`, `HookTimerImpl`.

### Dispatcher

- `Dispatcher` is the bridge to Task executors.
  - `Scheduler` must not know about the executors other than this `Dispatcher` interface.
  - Dispatch method blocks until at least one worker is available, and then returns a channel through which the result of Task execution would be notified.
- Currently only in-memory WorkerPool backed Dispatcher is implemented in ./dispatcher as `WorkerPoolDispatcher`.
  - It limits number of Task worked on concurrently.
  - `Dispatcher` could send tasks over network if you really want to do it; You WILL implement it your own.
    - This project probably will not provide an implementation, as it is not the goal of this project.

### WorkRegistry

- TBD...

### re-scheduler

- TBD...
  - cron-like, interval, and conditioned.

## TODOs

- Functionality
  - Add repository impls
    - [x] Add Gorm Repository
  - Add dispatcher impls
    - [ ] Add WorkerPool-Pool dispatcher.
  - Add WorkRegistry impls
    - [ ] Command-line work registry.
  - Add re-scheduler
    - [ ] cron like
    - [ ] interval
    - [ ] Add Task meta data.
    - [ ] Add Task deadline
- Refactor / Fix
  - [ ] HeapRepository need some more efforts.
    - [ ] Delete
  - [ ] Strip down GormCore to make it simpler and smaller interface.
  - [ ] Add tests for scheduler.
