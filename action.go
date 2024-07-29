package yotei

import "time"

// Action is the result of a task.
//
// It allows applying a logic based
// on the task's output.
type Action func(scheduler *Scheduler, task *Task)

func (action Action) Remove() Action {
	return func(scheduler *Scheduler, task *Task) {
		scheduler.Remove(task)

		if action != nil {
			action(scheduler, task)
		}
	}
}

func (action Action) Continue() Action {
	return func(scheduler *Scheduler, task *Task) {
		if action != nil {
			action(scheduler, task)
		}
	}
}

func (action Action) Add(tasks ...*Task) Action {
	return func(scheduler *Scheduler, task *Task) {
		scheduler.Add(tasks...)

		if action != nil {
			action(scheduler, task)
		}
	}
}

// Continue keeps the task in the scheduler.
func Continue() Action {
	return Action(nil)
}

// Retry will remove the task from the scheduler
// and add it back again after the given duration.
//
// # Important
//
// This does not mean the task will begin executing
// after the duration exceeds but rather that the
// task will be in the scheduler again after that time.
func Retry(duration time.Duration) Action {
	return func(scheduler *Scheduler, task *Task) {
		scheduler.Remove(task)

		go func() {
			time.Sleep(duration)
			scheduler.Add(task)
		}()
	}
}

// Done removes the task from the scheduler
func Done() Action {
	return func(scheduler *Scheduler, task *Task) {
		scheduler.Remove(task)
	}
}
