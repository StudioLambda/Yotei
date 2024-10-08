package yotei

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"math/rand"
	"runtime"
	"slices"
	"sync"
)

// Scheduler is the main structure to handler
// task scheduling in yotei.
//
// Use [NewScheduler] to create a new scheduler.
type Scheduler struct {
	workers uint64
	tasks   Tasks
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	logger  *slog.Logger
	mutex   sync.Mutex
}

// WorkersNumCPUs uses the number of CPU cores of the computer.
// as the number of workers.
var (
	// SingleWorker fires a scheduler with just one worker.
	SingleWorker uint64 = 1

	// NumCPUsWorkers fires a scheduler with [runtime.NumCPU] workers.
	NumCPUsWorkers uint64 = uint64(runtime.NumCPU())
)

var (
	// DefaultLogger is the default logger for the yotei scheduler.
	DefaultLogger *slog.Logger = slog.Default()

	// SilentLogger is a silent logger for the yotei scheduler.
	SilentLogger *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
)

// NewScheduler creates a new scheduler with the given workers and logger.
//
// If workers is `0` or [WorkersNumCPUs], the number of CPUs in the machine
// is used, as acording to [runtime.NumCPU].
//
// If the logger is `nil` or [DefaultLogger], the [slog.Default] will be used.
//
// # Example
//
//	yotei.NewScheduler(
//		yotei.WorkersNumCPUs,
//		yotei.DefaultLogger,
//	)
func NewScheduler(workers uint64, logger *slog.Logger) *Scheduler {
	if workers == 0 {
		workers = NumCPUsWorkers
	}

	if logger == nil {
		logger = DefaultLogger
	}

	return &Scheduler{
		workers: workers,
		logger:  logger,
	}
}

// Add appends a task into the scheduler. If the task
// was already in the scheduler it will ignore it.
func (scheduler *Scheduler) Add(tasks ...Tasker) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	for _, task := range tasks {
		for _, t := range scheduler.tasks {
			if t == task {
				continue
			}
		}

		scheduler.tasks = append(scheduler.tasks, task)
	}
}

// Has returns true if the given task is currently
// in the scheduler.
func (scheduler *Scheduler) Has(task Tasker) bool {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	for _, t := range scheduler.tasks {
		if t == task {
			return true
		}
	}

	return false
}

// Remove deletes the given tasks from the scheduler.
func (scheduler *Scheduler) Remove(tasks ...Tasker) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	for i, t := range scheduler.tasks {
		for _, task := range tasks {
			if t == task {
				scheduler.tasks = append(scheduler.tasks[:i], scheduler.tasks[i+1:]...)
			}
		}
	}
}

func (scheduler *Scheduler) next() Tasker {
	if !scheduler.mutex.TryLock() {
		return nil
	}

	defer scheduler.mutex.Unlock()

	tasks := scheduler.tasks.Unlocked()
	weight := tasks.Weight()

	if weight == 0 {
		return nil
	}

	pick := rand.Uint64() % weight
	current := uint64(0)

	for _, task := range tasks {
		current += task.Weight()

		if pick < current {
			if !task.IsConcurrent() {
				task.Lock()
			}

			return task
		}
	}

	return nil
}

func (scheduler *Scheduler) handle(ctx context.Context, task Tasker) {
	if action := task.Handle(ctx); action != nil {
		action(scheduler, task)
	}
}

func (scheduler *Scheduler) handleTasker(task Tasker) {
	defer func() {
		if !task.IsConcurrent() {
			task.Unlock()
		}
	}()

	if duration := task.Duration(); duration > 0 {
		ctx, cancel := context.WithTimeout(scheduler.ctx, duration)
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
		case <-scheduler.ctx.Done():
			return
		default:
			if task := scheduler.next(); task != nil {
				scheduler.handleTasker(task)
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
func (scheduler *Scheduler) Start() {
	if scheduler.IsRunning() {
		return
	}

	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	scheduler.ctx, scheduler.cancel = context.WithCancel(context.Background())

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

	scheduler.cancel()
	scheduler.wg.Wait()
	scheduler.tasks = nil
	scheduler.ctx = nil
	scheduler.cancel = nil
}

// IsRunning determines if the scheduler is
// currently running or not.
func (scheduler *Scheduler) IsRunning() bool {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	return scheduler.ctx != nil
}

// Snapshot returns the current scheduler tasks as an iterator.
//
// When creating the iterator, the actual tasks are freezed
// from the moment this function called to ensure it
// is concurrent safe.
func (scheduler *Scheduler) Snapshot() iter.Seq[Tasker] {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	return slices.Values(slices.Clone(scheduler.tasks))
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
