//go:build integration

package micron_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
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
	id    int
	count atomic.Uint32

	err error
}

func (r *testRunner) Run(context.Context) error {
	r.count.Add(1)

	return r.err
}

func TestCron(t *testing.T) {
	logger := log.New(nil,
		log.WithLevel(slog.LevelDebug),
		log.WithTraceContext(true))

	testErr := errors.New("test error")
	runner1 := &testRunner{id: 1, count: atomic.Uint32{}}
	runner2 := &testRunner{id: 2, count: atomic.Uint32{}}
	runner3 := &testRunner{id: 3, count: atomic.Uint32{}, err: testErr}

	everytime := "* * * * * *"
	everyhalfmin := "0/2 * * * * *"
	twoMinEven := "0/2 * * * * *"
	twoMinOdd := "1/2 * * * * *"
	defaultDur := 1005 * time.Millisecond

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
			wants: map[int]int{1: 1, 2: 1},
		},
		{
			name: "TwoExecsTwoRunners",
			execMap: map[string][]executor.Runner{
				twoMinEven: {runner1},
				twoMinOdd:  {runner2},
			},
			dur:   2100 * time.Millisecond,
			wants: map[int]int{1: 1, 2: 1},
		},
		{
			name: "TwoExecsTwoRunnersLongShort",
			execMap: map[string][]executor.Runner{
				everyhalfmin: {runner1},
				everytime:    {runner2},
			},
			dur:   10315 * time.Millisecond,
			wants: map[int]int{1: 5, 2: 10},
		},
		{
			name: "TwoExecsOffsetFrequency",
			execMap: map[string][]executor.Runner{
				everytime: {runner1},
				twoMinOdd: {runner2},
			},
			dur:   2100 * time.Millisecond,
			wants: map[int]int{1: 2, 2: 1},
		},
		{
			name: "OneExecWithError",
			execMap: map[string][]executor.Runner{
				everytime: {runner3},
			},
			dur:   defaultDur,
			wants: map[int]int{3: 1},
			err:   testErr,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			results := make(map[int]int, len(testcase.wants))
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

					for _, runners := range testcase.execMap {
						for _, runner := range runners {
							if r, ok := runner.(*testRunner); ok {
								results[r.id] = int(r.count.Load())

								r.count.Store(0)
							}
						}
					}

					for k, v := range testcase.wants {
						res, ok := results[k]

						is.True(t, ok)
						is.Equal(t, v, res)
					}

					return
				case err = <-errCh:
					is.True(t, errors.Is(err, testcase.err))
				}
			}
		})
	}
}

func TestFillErrorBuffer(t *testing.T) {
	h := slog.NewJSONHandler(os.Stderr, nil)
	testErr := errors.New("test error")
	runner1 := &testRunner{id: 1, count: atomic.Uint32{}, err: testErr}
	dur := 2100 * time.Millisecond
	wants := 2

	exec, err := micron.NewExecutor("test_exec", []executor.Runner{runner1},
		executor.WithSchedule("* * * * * *", time.Local),
		executor.WithLogHandler(h),
	)
	is.Empty(t, err)

	sel, err := micron.NewSelector(
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
			result := runner1.count.Load()
			is.Equal(t, wants, int(result))

			return
		case err = <-errCh:
			is.True(t, errors.Is(err, testErr))
		}
	}
}
