//go:build integration

package selector_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/v3/executor"
	"github.com/zalgonoise/micron/v3/metrics"
	"github.com/zalgonoise/micron/v3/selector"
)

type testRunner struct {
	id    int
	count atomic.Uint32

	err error
}

func (r *testRunner) Run(context.Context) error {
	r.count.Add(1)

	return r.err
}

//nolint:gocognit // ignore cyclomatic complexity in integration tests, for its extended setup logic
func TestSelector(t *testing.T) {
	h := slog.NewJSONHandler(os.Stderr, nil)

	runner1 := &testRunner{id: 1, count: atomic.Uint32{}}
	runner2 := &testRunner{id: 2, count: atomic.Uint32{}}

	everytime := "* * * * * *"
	everytime2 := "0-59 * * * * *"
	twoMinEven := "0/2 * * * * *"
	twoMinOdd := "1/2 * * * * *"
	defaultDur := 2010 * time.Millisecond

	for _, testcase := range []struct {
		name    string
		execMap map[string][]executor.Runner // cron string : runners
		dur     time.Duration
		wants   map[int]int
		err     error
	}{
		{
			name: "SingleExecTwoRunners",
			execMap: map[string][]executor.Runner{
				everytime: {runner1, runner2},
			},
			dur:   defaultDur,
			wants: map[int]int{1: 2, 2: 2},
		},
		{
			name: "TwoExecsTwoRunners",
			execMap: map[string][]executor.Runner{
				twoMinEven: {runner1},
				twoMinOdd:  {runner2},
			},
			dur:   defaultDur,
			wants: map[int]int{1: 1, 2: 1},
		},
		{
			name: "TwoExecsOffsetFrequency",
			execMap: map[string][]executor.Runner{
				everytime: {runner1},
				twoMinOdd: {runner2},
			},
			dur:   defaultDur,
			wants: map[int]int{1: 2, 2: 1},
		},
		{
			name: "TwoExecsSameSchedule",
			execMap: map[string][]executor.Runner{
				everytime:  {runner1},
				everytime2: {runner2},
			},
			dur:   defaultDur,
			wants: map[int]int{1: 2, 2: 2},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			defer func() {
				runner1.count.Store(0)
				runner2.count.Store(0)
			}()

			execs := make([]selector.Executor, 0, len(testcase.execMap))

			var n int
			for cron, runners := range testcase.execMap {
				exec, err := executor.New(fmt.Sprintf("%d", n), runners,
					executor.WithSchedule(cron, time.Local),
					executor.WithLogHandler(h),
				)
				is.Empty(t, err)

				execs = append(execs, exec)
				n++
			}

			selectorOpts := []cfg.Option[*selector.Config]{
				selector.WithExecutors(execs...),
				selector.WithLogHandler(h),
			}

			var (
				sel *selector.Selector
				err error
			)

			sel, err = selector.New(selectorOpts...)

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

					results := make(map[int]int, 2)
					results[runner1.id] = int(runner1.count.Load())
					results[runner2.id] = int(runner2.count.Load())

					for k, v := range testcase.wants {
						res, ok := results[k]
						is.True(t, ok)

						is.Equal(t, v, res)
					}

					return
				}
			}
		})
	}
}

//nolint:gocognit // ignore cyclomatic complexity in integration tests, for its extended setup logic
func TestBlockingSelector(t *testing.T) {
	h := slog.NewJSONHandler(os.Stderr, nil)

	runner1 := &testRunner{id: 1, count: atomic.Uint32{}}
	runner2 := &testRunner{id: 2, count: atomic.Uint32{}}

	everytime := "* * * * * *"
	everytime2 := "0-59 * * * * *"
	twoMinEven := "0/2 * * * * *"
	twoMinOdd := "1/2 * * * * *"
	defaultDur := 2010 * time.Millisecond

	for _, testcase := range []struct {
		name    string
		execMap map[string][]executor.Runner // cron string : runners
		dur     time.Duration
		wants   map[int]int
		err     error
	}{
		{
			name: "SingleExecTwoRunners",
			execMap: map[string][]executor.Runner{
				everytime: {runner1, runner2},
			},
			dur:   defaultDur,
			wants: map[int]int{1: 2, 2: 2},
		},
		{
			name: "TwoExecsTwoRunners",
			execMap: map[string][]executor.Runner{
				twoMinEven: {runner1},
				twoMinOdd:  {runner2},
			},
			dur:   defaultDur,
			wants: map[int]int{1: 1, 2: 1},
		},
		{
			name: "TwoExecsOffsetFrequency",
			execMap: map[string][]executor.Runner{
				everytime: {runner1},
				twoMinOdd: {runner2},
			},
			dur: defaultDur,
			// id: 2 doesn't get executed (blocked by id: 1)
			wants: map[int]int{1: 2, 2: 0},
		},
		{
			name: "TwoExecsSameSchedule",
			execMap: map[string][]executor.Runner{
				everytime:  {runner1},
				everytime2: {runner2},
			},
			dur: defaultDur,
			// id: 2 doesn't get executed (blocked by id: 1)
			wants: map[int]int{1: 2, 2: 0},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			defer func() {
				runner1.count.Store(0)
				runner2.count.Store(0)
			}()

			execs := make([]selector.Executor, 0, len(testcase.execMap))

			var n int
			for cron, runners := range testcase.execMap {
				exec, err := executor.New(fmt.Sprintf("%d", n), runners,
					executor.WithSchedule(cron, time.Local),
					executor.WithLogHandler(h),
				)
				is.Empty(t, err)

				execs = append(execs, exec)
				n++
			}

			selectorOpts := []cfg.Option[*selector.Config]{
				selector.WithExecutors(execs...),
				selector.WithLogHandler(h),
			}

			var (
				sel *selector.BlockingSelector
				err error
			)

			sel, err = selector.NewBlockingSelector(selectorOpts...)

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

					results := make(map[int]int, 2)
					results[runner1.id] = int(runner1.count.Load())
					results[runner2.id] = int(runner2.count.Load())

					for k, v := range testcase.wants {
						res, ok := results[k]
						is.True(t, ok)

						is.Equal(t, v, res)
					}

					return
				}
			}
		})
	}
}

func TestNonBlocking(t *testing.T) {
	cron := "* * * * * *"
	h := slog.NewJSONHandler(os.Stderr, nil)
	testErr := errors.New("test error")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	exec, err := executor.New("test", []executor.Runner{
		executor.Runnable(func(ctx context.Context) error {
			<-time.After(100 * time.Millisecond)

			return testErr
		})},
		executor.WithSchedule(cron, time.Local),
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
