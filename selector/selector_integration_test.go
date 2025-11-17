//go:build integration

package selector_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/micron/v3"
	"github.com/zalgonoise/micron/v3/log"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/v3/executor"
	"github.com/zalgonoise/micron/v3/metrics"
	"github.com/zalgonoise/micron/v3/selector"
)

type testRunner struct {
	id    int
	count *atomic.Uint32

	err error
}

func (r *testRunner) Run(context.Context) error {
	r.count.Add(1)

	return r.err
}

//nolint:gocognit // ignore cyclomatic complexity in integration tests, for its extended setup logic
func TestSelector(t *testing.T) {
	logger := log.New(nil, log.AsJSON(), log.WithLevel(slog.LevelDebug))

	runner1 := &testRunner{id: 1, count: &atomic.Uint32{}}
	runner2 := &testRunner{id: 2, count: &atomic.Uint32{}}
	runnerReboot := &testRunner{id: 3, count: &atomic.Uint32{}}

	reboot := "@reboot"
	everytime := "* * * * * *"
	everytime2 := "0-59 * * * * *"
	twoMinOdd := "1/2 * * * * *"

	twoSecEven := "0/2 * * * * *"
	twoSecOdd := "1/2 * * * * *"
	defaultDur := 2030 * time.Millisecond

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
				twoSecEven: {runner1},
				twoSecOdd:  {runner2},
			},
			dur:   2 * defaultDur,
			wants: map[int]int{1: 2, 2: 2},
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
		{
			name: "TwoExecsEverytimeAndReboot",
			execMap: map[string][]executor.Runner{
				everytime: {runner1},
				reboot:    {runnerReboot},
			},
			dur:   defaultDur,
			wants: map[int]int{1: 2, 3: 1},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			runner1.count.Store(0)
			runner2.count.Store(0)
			runnerReboot.count.Store(0)

			execs := make([]selector.Executor, 0, len(testcase.execMap))

			var n int
			for cron, runners := range testcase.execMap {
				exec, err := executor.New(cron, runners,
					executor.WithSchedule(cron, time.Local),
					executor.WithLogger(logger),
					executor.WithTimeout(30*time.Second),
				)
				is.Empty(t, err)

				execs = append(execs, exec)
				n++
			}

			selectorOpts := []cfg.Option[*selector.Config]{
				selector.WithExecutors(execs...),
				selector.WithLogger(logger),
				selector.WithTimeout(30 * time.Second),
			}

			var (
				sel *selector.Selector
				err error
			)

			ctx, cancel := context.WithTimeout(context.Background(), testcase.dur)
			defer cancel()

			sel, err = selector.New(ctx, selectorOpts...)
			is.Empty(t, err)

			c, err := micron.New(
				micron.WithSelector(sel),
				micron.WithLogger(logger))
			is.Empty(t, err)

			// try to stick to zero seconds in tests as much as possible
			start := time.Now().Truncate(time.Second).Add(990 * time.Millisecond)
			t.Log("start: ", start.Format(time.RFC3339))
			time.Sleep(time.Until(start))

			c.Run(ctx)

			<-ctx.Done()
			if testcase.dur < time.Second {
				is.True(t, errors.Is(ctx.Err(), context.DeadlineExceeded))

				return
			}

			results := make(map[int]int, 3)
			results[runner1.id] = int(runner1.count.Load())
			results[runner2.id] = int(runner2.count.Load())
			results[runnerReboot.id] = int(runnerReboot.count.Load())

			for k, v := range testcase.wants {
				res, ok := results[k]
				is.True(t, ok)

				is.Equal(t, v, res)
			}
		})
	}
}

//nolint:gocognit // ignore cyclomatic complexity in integration tests, for its extended setup logic
func TestBlockingSelector(t *testing.T) {
	logger := log.New(nil, log.AsJSON(), log.WithLevel(slog.LevelDebug))

	runner1 := &testRunner{id: 1, count: &atomic.Uint32{}}
	runner2 := &testRunner{id: 2, count: &atomic.Uint32{}}
	runnerReboot := &testRunner{id: 3, count: &atomic.Uint32{}}

	reboot := "@reboot"
	everytime := "* * * * * *"
	everytime2 := "0-59 * * * * *"
	twoMinOdd := "1/2 * * * * *"

	twoSecEven := "0/2 * * * * *"
	twoSecOdd := "1/2 * * * * *"
	defaultDur := 2030 * time.Millisecond

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
				twoSecEven: {runner1},
				twoSecOdd:  {runner2},
			},
			dur:   2 * defaultDur,
			wants: map[int]int{1: 2, 2: 2},
		},
		{
			name: "TwoExecsOffsetFrequency",
			execMap: map[string][]executor.Runner{
				everytime: {runner1},
				twoMinOdd: {runner2},
			},
			dur:   2 * defaultDur,
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
		{
			name: "TwoExecsEverytimeAndReboot",
			execMap: map[string][]executor.Runner{
				everytime: {runner1},
				reboot:    {runnerReboot},
			},
			dur:   2 * defaultDur,
			wants: map[int]int{1: 2, 3: 1},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			runner1.count.Store(0)
			runner2.count.Store(0)
			runnerReboot.count.Store(0)

			execs := make([]selector.Executor, 0, len(testcase.execMap))

			var n int
			for cron, runners := range testcase.execMap {
				exec, err := executor.New(cron, runners,
					executor.WithSchedule(cron, time.Local),
					executor.WithLogger(logger),
					executor.WithTimeout(30*time.Second),
				)
				is.Empty(t, err)

				execs = append(execs, exec)
				n++
			}

			selectorOpts := []cfg.Option[*selector.Config]{
				selector.WithExecutors(execs...),
				selector.WithLogger(logger),
				selector.WithTimeout(30 * time.Second),
			}

			var (
				sel *selector.BlockingSelector
				err error
			)

			ctx, cancel := context.WithTimeout(context.Background(), testcase.dur)
			defer cancel()

			sel, err = selector.NewBlockingSelector(selectorOpts...)
			is.Empty(t, err)

			c, err := micron.New(
				micron.WithSelector(sel),
				micron.WithLogger(logger))
			is.Empty(t, err)

			// try to stick to zero seconds in tests as much as possible
			start := time.Now().Truncate(time.Second).Add(990 * time.Millisecond)
			t.Log("start: ", start.Format(time.RFC3339))
			time.Sleep(time.Until(start))

			c.Run(ctx)

			<-ctx.Done()
			if testcase.dur < time.Second {
				is.True(t, errors.Is(ctx.Err(), context.DeadlineExceeded))

				return
			}

			results := make(map[int]int, 3)
			results[runner1.id] = int(runner1.count.Load())
			results[runner2.id] = int(runner2.count.Load())
			results[runnerReboot.id] = int(runnerReboot.count.Load())

			for k, v := range testcase.wants {
				res, ok := results[k]
				is.True(t, ok)

				is.Equal(t, v, res)
			}
		})
	}
}

func TestNonBlocking(t *testing.T) {
	cron := "* * * * * *"
	h := slog.NewJSONHandler(os.Stderr, nil)
	testErr := errors.New("test error")

	exec, err := executor.New("test", []executor.Runner{
		executor.Runnable(func(ctx context.Context) error {
			<-time.After(50 * time.Millisecond)

			return testErr
		})},
		executor.WithSchedule(cron, time.Local),
		executor.WithLogHandler(h),
	)
	is.Empty(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	sel, err := selector.New(ctx,
		selector.WithExecutors(exec),
		selector.WithLogHandler(h),
		selector.WithMetrics(metrics.NoOp()),
		selector.WithTrace(noop.NewTracerProvider().Tracer("test")),
		selector.WithTimeout(100*time.Millisecond),
	)
	is.Empty(t, err)

	c, err := micron.New(micron.WithSelector(sel))
	is.Empty(t, err)

	c.Run(ctx)
}
