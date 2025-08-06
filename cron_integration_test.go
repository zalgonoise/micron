//go:build integration

package micron_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/zalgonoise/micron/v3/log"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/v3"
	"github.com/zalgonoise/micron/v3/executor"
	"github.com/zalgonoise/micron/v3/metrics"
	"github.com/zalgonoise/micron/v3/selector"
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
	logger := log.New(nil,
		log.WithLevel(slog.LevelDebug),
		log.WithTraceContext(true))

	testErr := errors.New("test error")
	values := make(chan int)
	runner1 := testRunner{v: 1, ch: values}
	runner2 := testRunner{v: 2, ch: values}
	runner3 := testRunner{v: 3, ch: values, err: testErr}

	everytime := "* * * * * *"
	everymin := "0 * * * * *"
	everyhalfmin := "0/2 * * * * *"
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
				everytime: {runner1, runner2},
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
			name: "TwoExecsTwoRunnersLongShort",
			execMap: map[string][]executor.Runner{
				everyhalfmin: {runner1},
				everymin:     {runner2},
			},
			dur: time.Minute + 15*time.Second,
			wants: []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 2},
		},
		{
			name: "TwoExecsOffsetFrequency",
			execMap: map[string][]executor.Runner{
				everytime: {runner1},
				twoMinOdd: {runner2},
			},
			dur:   2100 * time.Millisecond,
			wants: []int{1, 1, 2},
		},
		{
			name: "OneExecWithError",
			execMap: map[string][]executor.Runner{
				everytime: {runner3},
			},
			dur:   defaultDur,
			wants: []int{3},
			err:   testErr,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			results := make([]int, 0, len(testcase.wants))
			execs := make([]selector.Executor, 0, len(testcase.execMap))

			var n int
			for cron, runners := range testcase.execMap {
				exec, err := executor.New(fmt.Sprintf("%d", n), runners,
					executor.WithSchedule(cron, time.Local),
					executor.WithLogger(logger),
				)
				is.Empty(t, err)

				execs = append(execs, exec)
				n++
			}

			sel, err := selector.New(
				selector.WithExecutors(execs...),
				selector.WithLogger(logger),
			)
			is.Empty(t, err)

			c, err := micron.New(
				micron.WithSelector(sel),
				micron.WithLogger(logger),
				micron.WithErrorBufferSize(5),
				micron.WithMetrics(metrics.NoOp()),
				micron.WithTrace(noop.NewTracerProvider().Tracer("test")),
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

	exec, err := executor.New("test_exec", []executor.Runner{runner1},
		executor.WithSchedule("* * * * * *", time.Local),
		executor.WithLogHandler(h),
	)
	is.Empty(t, err)

	sel, err := selector.New(
		selector.WithExecutors(exec),
		selector.WithLogHandler(h),
	)
	is.Empty(t, err)

	c, err := micron.New(
		micron.WithSelector(sel),
		micron.WithLogHandler(h),
		micron.WithErrorBufferSize(0),
		micron.WithMetrics(metrics.NoOp()),
		micron.WithTrace(noop.NewTracerProvider().Tracer("test")),
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
