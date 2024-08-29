package yotei

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// Task is a sequential task executionable in the [yotei.Scheduler].
//
// Use [NewTask] to create a new sequential task.
type Task struct {
	handler    Handler
	weight     atomic.Uint64
	duration   atomic.Int64
	locked     atomic.Bool
	concurrent atomic.Bool
}

// NewTask creates a new sequential task with the given handler.
func NewTask(handler Handler) *Task {
	if handler == nil {
		panic("no task handler defined. please ensure the task handler is not nil")
	}

	task := &Task{
		handler: handler,
	}

	task.weight.Store(1)
	task.duration.Store(int64(DurationUnlimited))
	task.concurrent.Store(false)

	return task
}

func (task *Task) Lock() {
	task.locked.Store(true)
}

func (task *Task) Unlock() {
	task.locked.Store(false)
}

func (task *Task) Concurrent(value bool) *Task {
	task.concurrent.Store(value)

	return task
}

func (task *Task) IsLocked() bool {
	return task.locked.Load()
}

func (task *Task) IsConcurrent() bool {
	return task.concurrent.Load()
}

func (task *Task) Lasts(duration time.Duration) *Task {
	task.duration.Store(int64(duration))

	return task
}

func (task *Task) Duration() time.Duration {
	return time.Duration(task.duration.Load())
}

func (task *Task) Weights(weight uint64) *Task {
	task.weight.Store(weight)

	return task
}

func (task *Task) Weight() uint64 {
	return task.weight.Load()
}

func (task *Task) Handle(ctx context.Context) Action {
	if task.handler == nil {
		panic("no task handler defined. please ensure the task handler is not nil")
	}

	return task.handler.Handle(ctx)
}

// String returns a string representation of a task.
func (task *Task) String() string {
	return fmt.Sprintf(
		"Task{weight=%d, duration=%s, is_concurrent=%t, is_locked=%t}",
		task.Weight(),
		task.Duration(),
		task.IsConcurrent(),
		task.IsLocked(),
	)
}
