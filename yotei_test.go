package yotei_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/studiolambda/yotei"
)

type CounterHandler atomic.Uint64

func (counter *CounterHandler) Handle(ctx context.Context) yotei.Action {
	_ = (*atomic.Uint64)(counter).Add(1)

	return yotei.Continue()
}

func (counter *CounterHandler) Count() uint64 {
	return (*atomic.Uint64)(counter).Load()
}

func TestThreeTasks(t *testing.T) {
	scheduler := yotei.NewScheduler(
		yotei.WorkersNumCPUs,
		yotei.DefaultLogger,
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

func TestItDoesNotRunLockedTasks(t *testing.T) {
	scheduler := yotei.NewScheduler(
		12,
		yotei.DefaultLogger,
	)

	counter1 := &CounterHandler{}
	counter2 := &CounterHandler{}

	tasks := yotei.Tasks{
		yotei.
			NewTask(counter1).
			Concurrent(false).
			Weights(10).
			Lasts(10 * time.Millisecond),
		yotei.
			NewTask(counter2).
			Concurrent(true).
			Weights(10).
			Lasts(10 * time.Millisecond),
	}

	scheduler.Add(tasks...)
	scheduler.Start()
	time.Sleep(100 * time.Millisecond)
	scheduler.Stop()

	t.Log(tasks[0], "->", counter1.Count())
	t.Log(tasks[1], "->", counter2.Count())

	if expected := uint64(10); counter1.Count() > expected {
		t.Fatalf(
			"counter1=%d should not be higher than expected=%d",
			counter1.Count(),
			expected,
		)
	}
}

func TestSequence(t *testing.T) {
	scheduler := yotei.NewScheduler(
		yotei.WorkersNumCPUs,
		yotei.DefaultLogger,
	)

	calls := make([]string, 0)

	var lastHandler yotei.HandlerFunc = func(context.Context) yotei.Action {
		calls = append(calls, "last")

		return yotei.Done()
	}

	last := yotei.NewTask(lastHandler)

	var nextHandler yotei.HandlerFunc = func(context.Context) yotei.Action {
		calls = append(calls, "next")

		return yotei.Done().ThenAdd(last)
	}

	next := yotei.NewTask(nextHandler)

	var initialHandler yotei.HandlerFunc = func(context.Context) yotei.Action {
		calls = append(calls, "initial")

		return yotei.Done().ThenAdd(next)
	}

	initial := yotei.NewTask(initialHandler)

	scheduler.Add(initial)
	scheduler.Start()
	time.Sleep(10 * time.Millisecond)
	scheduler.Stop()

	if expected := 3; len(calls) != expected {
		t.Fatalf("expected len(calls)=%d but got %d", expected, len(calls))
	}

	if expected := "initial"; calls[0] != expected {
		t.Fatalf("expected calls[0]=%s but got %s", expected, calls[0])
	}

	if expected := "next"; calls[1] != expected {
		t.Fatalf("expected calls[1]=%s but got %s", expected, calls[1])
	}

	if expected := "last"; calls[2] != expected {
		t.Fatalf("expected calls[2]=%s but got %s", expected, calls[2])
	}
}
