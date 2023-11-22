//go:build integration

package selector_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

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

func TestSelector(t *testing.T) {
	h := slog.NewJSONHandler(os.Stderr, nil)

	values := make(chan int)
	runner1 := testRunner{v: 1, ch: values}
	runner2 := testRunner{v: 2, ch: values}

	cron := "* * * * * *"
	cron2 := "0-59 * * * * *"
	twoMinEven := "0/2 * * * * *"
	twoMinOdd := "1/2 * * * * *"
	defaultDur := 2010 * time.Millisecond

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
				cron: {runner1, runner2},
			},
			dur:   defaultDur,
			wants: []int{1, 1, 2, 2},
		},
		{
			name: "TwoExecsTwoRunners",
			execMap: map[string][]executor.Runner{
				twoMinEven: {runner1},
				twoMinOdd:  {runner2},
			},
			dur:   defaultDur * 2,
			wants: []int{1, 1, 2, 2},
		},
		{
			name: "TwoExecsOffsetFrequency",
			execMap: map[string][]executor.Runner{
				cron:      {runner1},
				twoMinOdd: {runner2},
			},
			dur:   defaultDur,
			wants: []int{1, 1, 2},
		},
		{
			name: "TwoExecsSameSchedule",
			execMap: map[string][]executor.Runner{
				cron:  {runner1},
				cron2: {runner2},
			},
			dur:   defaultDur,
			wants: []int{1, 1, 2, 2},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			testFunc := func(t *testing.T, withBlock bool) {
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

				selectorOpts := []cfg.Option[selector.Config]{
					selector.WithExecutors(execs...),
					selector.WithLogHandler(h),
				}

				if withBlock {
					selectorOpts = append(selectorOpts, selector.WithBlock())
				}

				sel, err := selector.New(selectorOpts...)

				is.Empty(t, err)

				ctx, cancel := context.WithTimeout(context.Background(), testcase.dur)

				go func(t *testing.T, err error) {
					defer cancel()

					for {
						select {
						case <-ctx.Done():
							return
						default:
						}

						selErr := sel.Next(ctx)
						if !errors.Is(selErr, err) && !errors.Is(selErr, context.DeadlineExceeded) {
							t.Error(selErr)
						}
					}
				}(t, testcase.err)

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
					case v := <-values:
						results = append(results, v)
					}
				}
			}

			t.Run("WithBlock", func(t *testing.T) {
				testFunc(t, true)
			})

			t.Run("NonBlocking", func(t *testing.T) {
				testFunc(t, false)
			})
		})
	}
}

func TestNonBlocking(t *testing.T) {
	cronString := "* * * * * *"
	h := slog.NewJSONHandler(os.Stderr, nil)
	testErr := errors.New("test error")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	exec, err := executor.New("test",
		executor.WithSchedule(cronString),
		executor.WithLocation(time.Local),
		executor.WithRunners(executor.Runnable(func(ctx context.Context) error {
			<-time.After(100 * time.Millisecond)

			return testErr
		})),
		executor.WithLogHandler(h),
	)
	is.Empty(t, err)

	sel, err := selector.New(
		selector.WithExecutors(exec),
		selector.WithLogHandler(h),
		selector.WithMetrics(metrics.NoOp()),
		selector.WithTrace(noop.NewTracerProvider().Tracer("test")),
		selector.WithTimeout(70*time.Millisecond),
	)
	is.Empty(t, err)

	errCh := make(chan error)

	go func() {
		errCh <- sel.Next(ctx)
	}()

	select {
	case <-ctx.Done():
		t.Error("context timeout with no error return")

		return
	case err = <-errCh:
		// error must be empty on this channel, but should be logged from the goroutine shortly after
		is.Empty(t, err)
	}

	<-ctx.Done()
}
