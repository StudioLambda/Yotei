# Yotei

Yotei is a powerful and flexible task scheduling library for Go, designed to handle a variety of concurrent tasks with ease. Utilizing a Weighted Round Robin (WRR) algorithm, this scheduler allows for efficient distribution of tasks among multiple workers, with support for custom task weights (priorities), sequential/concurrent tasks, and durations. Perfect for high-performance applications requiring fine-grained control over task execution.

## Features

- Weighted Round Robin (WRR) Scheduling: Efficiently distributes tasks among workers based on custom weights and priorities.
- Multiple Workers: Scale your task processing with support for multiple concurrent workers.
- Custom Task Weights: Assign specific weights to tasks to control their execution frequency and priority.
- Post-Task Actions: Define custom actions to be executed after each task is handled.
- Optional Task Locking: Ensure only a single worker processes a task at a time with optional task locking. Determines if the task is sequential or concurrent.
- Custom Task Duration: Specify exact durations for tasks, making them sleep if completed early or canceling their context if they run too long.

## Installation

```
go get github.com/studiolambda/yotei
```

## Documentation

[Official Documentation](https://pkg.go.dev/github.com/studiolambda/yotei)

## Example

```go
type CounterHandler atomic.Uint64

func (counter *CounterHandler) Handle(_ context.Context) yotei.Action {
	_ = (*atomic.Uint64)(counter).Add(1)

	return yotei.Continue()
}

func (counter *CounterHandler) Count() uint64 {
	return (*atomic.Uint64)(counter).Load()
}

func TestThreeTasks(t *testing.T) {
	scheduler := yotei.NewScheduler(
		yotei.NumCPUsWorkers,
		yotei.SilentLogger,
	)

	counter1 := &CounterHandler{}
	counter2 := &CounterHandler{}
	counter3 := &CounterHandler{}

	tasks := yotei.Tasks{
		yotei.
			NewTask(counter1).
			Weights(10).
			Concurrent(true),
		yotei.
			NewTask(counter2).
			Weights(20).
			Concurrent(true),
		yotei.
			NewTask(counter3).
			Weights(30).
			Concurrent(true),
	}

	scheduler.Add(tasks...)
	scheduler.Start()
	time.Sleep(2 * time.Millisecond)
	scheduler.Stop()

	t.Log(tasks[0], "->", counter1.Count())
	t.Log(tasks[1], "->", counter2.Count())
	t.Log(tasks[2], "->", counter3.Count())

	if counter1.Count() > counter2.Count() {
		t.Fatalf(
			"counter1=%d should not be higher than counter2=%d",
			counter1.Count(),
			counter2.Count(),
		)
	}

	if counter2.Count() > counter3.Count() {
		t.Fatalf(
			"counter2=%d should not be higher than counter3=%d",
			counter2.Count(),
			counter3.Count(),
		)
	}
}
```
