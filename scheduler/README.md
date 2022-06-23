# Scheduler

In-memory scheduler.

## Needed Goroutines

Scheduler needs 2+n goroutines, 1 for dispatch loop 1 for canceller loop and n for workers.

## Execution Delay

The delay between scheduled time and actual work invocation is typically under 30 milli secs. But you do want to do your own benchmark at your own setup.

## Parts

### Task

`Task` is defined as a minimum set of data relevant to scheduling, dispatching and executing. `Task` has scheduled-time and work, which is function to be executed on scheduled time, and some internal state like cancelled or done.

`Task` will be stored in min-heap.

### Min-heap

Scheduler stores tasks to the min-heap. It is a priority queue that prioritize least scheduled time, meaning earlist is most. It relies on [std container/heap](https://pkg.go.dev/container/heap@go1.18.3) implementation, which means element addition and retrival is O(log n) where n = len of elements.

### TaskTaskTimer

TaskTimer is wrapper of min-heap.

It sets timer to min element when task push / pop.

Popped tasks are sent to Worker-s via a channel.

### Workers and WorkerPool

Worker is executor of tasks. Does work on single task at a time.

WorkerPool is, as its name says, a pool of Worker. It provides a way to dynamically increase and decrease workers. That number limits how many tasks can be worked on concurrently. Zero worker = no task can be sent on channel. So it should be at least 1.
