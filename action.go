package yotei

import "time"

// Action is the result of a task.
//
// It allows applying a logic based
// on the task's output.
type Action func(scheduler *Scheduler, task *Task)

// ThenAdd adds the given tasks to the scheduler
// when the action is run.
func (action Action) ThenAdd(tasks ...*Task) Action {
	return action.Then(func(scheduler *Scheduler, task *Task) {
		scheduler.Add(tasks...)
	})
}

// Then adds a new callback that will be executed when the action
// is run. The callback supports a specific action callback that
// gets the current scheduler and the executed task.
func (action Action) Then(callback Action) Action {
	return func(scheduler *Scheduler, task *Task) {
		if action != nil {
			action(scheduler, task)
		}

		callback(scheduler, task)
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
