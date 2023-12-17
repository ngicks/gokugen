# gokugen [![Godoc](https://godoc.org/github.com/ngicks/gokugen?status.svg)](https://godoc.org/github.com/ngicks/gokugen)

go+刻限(kokugen)

The term 刻限(kokugen) is a japanese word equivalent of appointed time, scheduled time, or due.

gokugen is set of interfaces and a scheduler implementation. Mainly focused on single-process scheduler.

## The original idea

The idea is based on [this article](https://qiita.com/kawasin73/items/7af6766c7898a656b1ee)(written in japanese).

In the article, it uses binary heap to store and sort task by priority(earliest scheduled is most), and call associated function in a worker pool.

In this project, things are broken up so that those can be swapped.

## Arch

![arch.drawio.svg](./arch.drawio.svg)

Things are decomposed to 3 elements.

- Model definition
- Repository
- Dispatcher

## Implementations

### Scheduler

- `Scheduler` is a state machine that:
  - wait for the next task
  - fetch the next task
  - dispatch the task through dispatcher
  - once the task is complete, then record that to Repository.

### ObservableRepository

- `ObservableRepository` is a combination of `Repository` interface and `Observer` interface.
  - It provides the typical Repository interface, and
  - It provides special search capability; GetNext() returns the next scheduled Task.
  - Also it watches task addition, completion and cancellation. It resets the timer to the next scheduled time if that is changed.
- Currently the project has 2 implementation in ./repository.
  - `*InMemoryRepository`: in-memory Repository
    - It is backed by `map` and `container/heap`. The complexity of Task addition and removal is `O(log n)`.
  - `*EntRepository`: persistent Repository using the entity frame work; `entgo.io/ent`.

### Dispatcher

- `Dispatcher` is the bridge to Task executors.
  - `Scheduler` must not know about the executors other than this `Dispatcher` interface.
  - Dispatch method blocks until at least one worker is available, and then returns a channel through which the result of Task execution would be notified.
- Currently only in-memory WorkerPool backed Dispatcher is implemented in ./dispatcher as `WorkerPoolDispatcher`.
  - It limits number of Task worked on concurrently.
  - `Dispatcher` could send tasks over network if you really want to do it; You WILL implement it your own.
    - This project probably will not provide an implementation, as it is not the goal of this project.
