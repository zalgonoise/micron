//go:build integration

package cron_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron"
	"github.com/zalgonoise/micron/executor"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/selector"
)

type testRunner struct {
	v  int
	ch chan<- int

	err error
}

func (r testRunner) Run(context.Context) error {
	r.ch <- r.v

	return r.err
}

func TestCron(t *testing.T) {
	h := slog.NewJSONHandler(os.Stderr, nil)

	testErr := errors.New("test error")
	values := make(chan int)
	runner1 := testRunner{v: 1, ch: values}
	runner2 := testRunner{v: 2, ch: values}
	runner3 := testRunner{v: 3, ch: values, err: testErr}

	cronString := "* * * * * *"
	twoMinEven := "0/2 * * * * *"
	twoMinOdd := "1/2 * * * * *"
	defaultDur := 1005 * time.Millisecond

	for _, testcase := range []struct {
		name    string
		execMap map[string][]executor.Runner // cron string : runners
		dur     time.Duration
		wants   []int
		err     error
	}{
		{
			name: "SingleExecTwoRunners",
			execMap: map[string][]executor.Runner{
				cronString: {runner1, runner2},
			},
			dur:   defaultDur,
			wants: []int{1, 2},
		},
		{
			name: "TwoExecsTwoRunners",
			execMap: map[string][]executor.Runner{
				twoMinEven: {runner1},
				twoMinOdd:  {runner2},
			},
			dur:   2100 * time.Millisecond,
			wants: []int{1, 2},
		},
		{
			name: "TwoExecsOffsetFrequency",
			execMap: map[string][]executor.Runner{
				cronString: {runner1},
				twoMinOdd:  {runner2},
			},
			dur:   2100 * time.Millisecond,
			wants: []int{1, 1, 2},
		},
		{
			name: "OneExecWithError",
			execMap: map[string][]executor.Runner{
				cronString: {runner3},
			},
			dur:   defaultDur,
			wants: []int{3},
			err:   testErr,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			results := make([]int, 0, len(testcase.wants))
			execs := make([]executor.Executor, 0, len(testcase.execMap))

			var n int
			for cronString, runners := range testcase.execMap {
				exec, err := executor.New(fmt.Sprintf("%d", n),
					executor.WithSchedule(cronString),
					executor.WithLocation(time.Local),
					executor.WithRunners(runners...),
					executor.WithLogHandler(h),
				)
				is.Empty(t, err)

				execs = append(execs, exec)
				n++
			}

			sel, err := selector.New(
				selector.WithExecutors(execs...),
				selector.WithLogHandler(h),
			)
			is.Empty(t, err)

			c, err := cron.New(
				cron.WithSelector(sel),
				cron.WithLogHandler(h),
				cron.WithErrorBufferSize(5),
				cron.WithMetrics(metrics.NoOp()),
				cron.WithTrace(noop.NewTracerProvider().Tracer("test")),
			)
			is.Empty(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), testcase.dur)
			defer cancel()

			errCh := c.Err()

			go c.Run(ctx)

			for {
				select {
				case <-ctx.Done():
					if testcase.dur < time.Second {
						is.True(t, errors.Is(ctx.Err(), context.DeadlineExceeded))

						return
					}

					slices.Sort(results)
					is.EqualElements(t, testcase.wants, results)

					return
				case err = <-errCh:
					is.True(t, errors.Is(err, testcase.err))
				case v := <-values:
					t.Log("received", v)

					results = append(results, v)
				}
			}
		})
	}
}

func TestFillErrorBuffer(t *testing.T) {
	h := slog.NewJSONHandler(os.Stderr, nil)
	testErr := errors.New("test error")
	values := make(chan int)
	runner1 := testRunner{v: 1, ch: values, err: testErr}
	dur := 2100 * time.Millisecond
	results := make([]int, 0, 2)
	wants := []int{1, 1}

	exec, err := executor.New("test_exec",
		executor.WithSchedule("* * * * * *"),
		executor.WithLocation(time.Local),
		executor.WithRunners(runner1),
		executor.WithLogHandler(h),
	)
	is.Empty(t, err)

	sel, err := selector.New(
		selector.WithExecutors(exec),
		selector.WithLogHandler(h),
	)
	is.Empty(t, err)

	c, err := cron.New(
		cron.WithSelector(sel),
		cron.WithLogHandler(h),
		cron.WithErrorBufferSize(0),
		cron.WithMetrics(metrics.NoOp()),
		cron.WithTrace(noop.NewTracerProvider().Tracer("test")),
	)
	is.Empty(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	errCh := c.Err()

	go c.Run(ctx)

	for {
		select {
		case <-ctx.Done():
			slices.Sort(results)
			is.EqualElements(t, wants, results)

			return
		case err = <-errCh:
			is.True(t, errors.Is(err, testErr))
		case v := <-values:
			t.Log("received", v)

			results = append(results, v)
		}
	}
}
