package yotei

import (
	"time"
)

type Tasker interface {
	Handler
	Duration() time.Duration
	Weight() uint64
	Lock()
	Unlock()
	IsLocked() bool
	IsConcurrent() bool
}

// A list of actionable tasks
type Tasks []Tasker

// Determines that the task can take unlimited duration.
var DurationUnlimited time.Duration = 0

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
