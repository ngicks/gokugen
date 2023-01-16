# gokugen [![Godoc](https://godoc.org/github.com/ngicks/gokugen?status.svg)](https://godoc.org/github.com/ngicks/gokugen)

go+刻限(kokugen)

The term 刻限(kokugen) is a japanese word equivalent of appointed time, scheduled time, or due.

gokugen is set of interfaces and a scheduler implementation.

## Idea

The idea is based on [this article](https://qiita.com/kawasin73/items/7af6766c7898a656b1ee)(written in japanese).

This repository breaks up implementation into pieces, adds slight improvements, and recomposes them.

## Arch

![arch.drawio.svg](./arch.drawio.svg)

### Scheduler

- `Scheduler` runs a loop in which it waits for the next scheduled Task stored in `Repository`, and when the next scheduled time passed, it dispatches Tasks to workers through `Dispatcher`.
- It also runs a loop that serializes updates to the tasks.
  - It aims to avoid race conditions in updates.
- Scheduler also provide a way to observe its state changes and events via `LoopHooks`.
  - In `LoopHooks`, you may suppress non critical Repository specific errors, wait for `Repository` coming back from error state if it has temporality been down, or even retry Task update.

### TaskRepository

- `TaskRepository` is a combination of `RepositoryLike` interface and `TimerLike` interface.
  - It provides the typical Repository interface, and
  - It provides special search capability; GetNext() returns the next scheduled Task.
  - Also it watches task addition, completion and cancellation. It resets the timer to the next scheduled time if that is changed.
- Currently only in-memory repository is implemented in ./repository as `HeapRepository`.
  - It is backed by `map` and `container/heap`. The complexity of Task addition and removal is `O(log n)`.

### Dispatcher

- `Dispatcher` is the bridge to Task executors.
  - `Scheduler` must not know about the executors other than this `Dispatcher` interface.
  - Dispatch method blocks until at least one worker is available, and then returns a channel through which the result of Task execution would be notified.
- Currently only in-memory WorkerPool backed Dispatcher is implemented in ./dispatcher as `WorkerPoolDispatcher`.
  - It limits number of Task worked on concurrently.
