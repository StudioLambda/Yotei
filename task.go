package yotei

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// Handler determines an interface of something
// that can be handled.
type Handler interface {
	// Handle is the callback to execute the task action.
	Handle(context.Context) Action
}

// Task is the the executionable action in the [Scheduler].
//
// A task must:
//   - Be handlable
//   - Have a weight
//   - Have a duration
//   - Be sequential or concurrent
//
// Use [NewTask] to create a new task.
type Task struct {
	handler    Handler
	weight     atomic.Uint64
	duration   atomic.Int64
	locked     atomic.Bool
	concurrent atomic.Bool
}

// A list of actionable tasks
type Tasks []*Task

var (
	DurationUnlimited time.Duration = 0
)

// NewTask creates a new task with the given handler.
//
// By default, a task:
//   - Has a weight of 1
//   - Has unlimited duration
//   - Is aequential
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

// Weight returns the sum of all he
// weights of all the tasks in the list.
func (tasks Tasks) Weight() uint64 {
	total := uint64(0)

	for _, task := range tasks {
		total += task.Weight()
	}

	return total
}

// Unlocked returns the tasks that are unlocked
func (tasks Tasks) Unlocked() Tasks {
	unlocked := make(Tasks, 0)

	for _, task := range tasks {
		if !task.IsLocked() {
			unlocked = append(unlocked, task)
		}
	}

	return unlocked
}

// Locked returns the tasks that are locked
func (tasks Tasks) Locked() Tasks {
	locked := make(Tasks, 0)

	for _, task := range tasks {
		if task.IsLocked() {
			locked = append(locked, task)
		}
	}

	return locked
}
