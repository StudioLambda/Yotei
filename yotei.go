package yotei

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"runtime"
	"sync"
)

// Scheduler is the main structure to handler
// task scheduling in yotei.
//
// Use [NewScheduler] to create a new scheduler.
type Scheduler struct {
	workers uint64
	tasks   Tasks
	quit    chan struct{}
	wg      sync.WaitGroup
	logger  *slog.Logger
	mutex   sync.Mutex
}

// WorkersNumCPUs uses the number of CPU cores of the computer.
// as the number of workers.
var WorkersNumCPUs uint64 = 0

// DefaultLogger is the default logger for the yotei scheduler.
var DefaultLogger *slog.Logger = nil

// Creates a new scheduler with the given workers and logger.
//
// If workers is `0` or [WorkersNumCPU], the number of CPUs in the machine
// is used, as acording to [runtime.NumCPU].
//
// If the logger is `nil` or [DefaultLogger], the [slog.Default] will be used.
//
// # Example
//
//	yotei.NewScheduler(
//		yotei.WorkersNumCPU,
//		yotei.DefaultLogger,
//	)
func NewScheduler(workers uint64, logger *slog.Logger) *Scheduler {
	if workers == 0 {
		workers = uint64(runtime.NumCPU())
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Scheduler{
		workers: workers,
		logger:  logger,
	}
}

// Adds a task into the scheduler. If the task
// was already in the scheduler it will do nothing.
func (scheduler *Scheduler) Add(tasks ...*Task) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	for _, task := range tasks {
		for _, t := range scheduler.tasks {
			if t == task {
				return
			}
		}

		scheduler.tasks = append(scheduler.tasks, task)
	}
}

// Has returns true if the given task is currently
// in the scheduler.
func (scheduler *Scheduler) Has(task *Task) bool {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	for _, t := range scheduler.tasks {
		if t == task {
			return true
		}
	}

	return false
}

// Removes a task from the scheduler. Since no tasks
// can be duplicated in the scheduler, Remove stops
// when a match is found.
func (scheduler *Scheduler) Remove(task *Task) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	for i, t := range scheduler.tasks {
		if t == task {
			scheduler.tasks = append(scheduler.tasks[:i], scheduler.tasks[i+1:]...)
			return
		}
	}
}

func (scheduler *Scheduler) next() *Task {
	if !scheduler.mutex.TryLock() {
		return nil
	}

	defer scheduler.mutex.Unlock()

	tasks := scheduler.tasks.Unlocked()
	weight := tasks.Weight()
	pick := rand.Uint64() % weight
	current := uint64(0)

	for _, task := range tasks {
		current += task.Weight()

		if pick < current {
			if task.IsSequential() {
				task.Lock()
			}

			return task
		}
	}

	return nil
}

func (scheduler *Scheduler) handle(ctx context.Context, task *Task) {
	if action := task.Handle(ctx); action != nil {
		action(scheduler, task)
	}
}

func (scheduler *Scheduler) handleTask(task *Task) {
	defer func() {
		if task.IsSequential() {
			task.Unlock()
		}
	}()

	if duration := task.Duration(); duration > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		go scheduler.handle(ctx, task)

		<-ctx.Done()

		return
	}

	scheduler.handle(context.Background(), task)
}

func (scheduler *Scheduler) worker() {
	defer scheduler.wg.Done()

	for {
		select {
		case <-scheduler.quit:
			return
		default:
			if task := scheduler.next(); task != nil {
				scheduler.handleTask(task)
				continue
			}
		}
	}
}

// Start begins executing the scheduler with the given set
// of tasks. If the scheduler was already running, it won't do anything.
//
// The scheduler spawns exactly [Scheduler.workers] workers, each in its own
// goroutine.
//
// If the task list contains no tasks to run (len == 0) no workers will be
// spawned, a warning will be emitted using the [Scheduler.logger] and
// the scheduler will remain in a running state.
func (scheduler *Scheduler) Start(tasks Tasks) {
	if scheduler.IsRunning() {
		return
	}

	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	scheduler.tasks = tasks
	scheduler.quit = make(chan struct{})

	scheduler.logger.Info(
		"starting scheduler",
		"workers", scheduler.workers,
		"tasks", len(scheduler.tasks),
	)

	if len(scheduler.tasks) == 0 {
		scheduler.logger.Warn("no tasks to execute")
		return
	}

	for i := uint64(0); i < scheduler.workers; i++ {
		scheduler.wg.Add(1)
		go scheduler.worker()
	}
}

// Stop makes a running scheduler halt its execution.
// All the workers shut down gracefully, completing their
// current tasks.
//
// A call to [Scheduler.Stop] waits for all the workers
// to exit.
//
// If the scheduler was not running, this does nothing.
func (scheduler *Scheduler) Stop() {
	if !scheduler.IsRunning() {
		return
	}

	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	scheduler.logger.Info("stopping scheduler")

	close(scheduler.quit)
	scheduler.wg.Wait()
	scheduler.tasks = nil
	scheduler.quit = nil
}

// IsRunning determines if the scheduler is
// currently running or not.
func (scheduler *Scheduler) IsRunning() bool {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	return scheduler.quit != nil
}

// String returns a string representation of a scheduler.
func (scheduler *Scheduler) String() string {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	return fmt.Sprintf(
		"Scheduler{is_running=%t, tasks=%s}",
		scheduler.IsRunning(),
		scheduler.tasks,
	)
}
