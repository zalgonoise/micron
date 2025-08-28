//go:build integration

package executor_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/v3/executor"
	"github.com/zalgonoise/micron/v3/log"
	"github.com/zalgonoise/micron/v3/metrics"
	"github.com/zalgonoise/micron/v3/schedule"
	"github.com/zalgonoise/micron/v3/schedule/cronlex"
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

func testRunnable(atom *atomic.Uint32, value uint32) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		atom.Add(value)

		return nil
	}
}

func TestNewExecutor(t *testing.T) {
	cron := "* * * * * *"
	s, err := cronlex.Parse(cron)
	is.Empty(t, err)

	sched, err := schedule.New(schedule.WithSchedule(s))
	is.Empty(t, err)

	for _, testcase := range []struct {
		name    string
		id      string
		runners []executor.Runner
		opts    []cfg.Option[*executor.Executable]
		err     error
	}{
		{
			name:    "DefaultID",
			runners: []executor.Runner{&testRunner{}},
			opts: []cfg.Option[*executor.Executable]{
				executor.WithSchedule(cron, time.Local),
			},
		},
		{
			name:    "CustomScheduler",
			runners: []executor.Runner{&testRunner{}},
			opts: []cfg.Option[*executor.Executable]{
				executor.WithScheduler(sched),
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			_, err := executor.New(testcase.id, testcase.runners, testcase.opts...)
			t.Log(err)
			is.True(t, errors.Is(err, testcase.err))
		})
	}
}

func TestExecutor(t *testing.T) {
	testErr := errors.New("test error")
	values := &atomic.Uint32{}
	runner1 := &testRunner{id: 1, count: atomic.Uint32{}}
	runner2 := &testRunner{id: 2, count: atomic.Uint32{}}
	runner3 := &testRunner{id: 3, count: atomic.Uint32{}, err: testErr}
	runnable := testRunnable(values, 4)

	cron := "* * * * * *"
	defaultDur := 1010 * time.Millisecond

	for _, testcase := range []struct {
		name    string
		dur     time.Duration
		runners []executor.Runner
		wants   map[int]int
		err     error
	}{
		{
			name:    "ContextCanceled",
			dur:     10 * time.Millisecond,
			runners: []executor.Runner{runner1, runner2},
			err:     context.DeadlineExceeded,
		},
		{
			name:    "TwoRunners",
			dur:     defaultDur,
			runners: []executor.Runner{runner1, runner2},
			wants:   map[int]int{1: 1, 2: 1},
		},
		{
			name:    "TwoRunnersAndARunnable",
			dur:     defaultDur,
			runners: []executor.Runner{runner1, runner2, executor.Runnable(runnable)},
			wants:   map[int]int{1: 1, 2: 1, 0: 4},
		},
		{
			name:    "NilRunnable",
			dur:     defaultDur,
			runners: []executor.Runner{executor.Runnable(nil)},
			wants:   map[int]int{},
		},
		{
			name:    "ErrorRunner",
			dur:     defaultDur,
			runners: []executor.Runner{runner3},
			wants:   map[int]int{},
			err:     testErr,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			defer func() {
				values.Store(0)
				runner1.count.Store(0)
				runner2.count.Store(0)
				runner3.count.Store(0)
			}()

			exec, err := executor.New(testcase.name, testcase.runners,
				executor.WithSchedule(cron, time.Local),
				executor.WithMetrics(metrics.NoOp()),
				executor.WithLogHandler(log.NoOp()),
				executor.WithLogger(slog.New(log.NoOp())),
				executor.WithTrace(noop.NewTracerProvider().Tracer("test")),
			)
			is.Empty(t, err)
			is.Equal(t, testcase.name, exec.ID())

			ctx, cancel := context.WithTimeout(context.Background(), testcase.dur)

			now := time.Now()

			go func(t *testing.T, err error) {
				defer cancel()

				_ = exec.Next(ctx, now)

				execErr := exec.Exec(ctx, now)
				is.True(t, errors.Is(execErr, err))
			}(t, testcase.err)

			for {
				select {
				case <-ctx.Done():
					if testcase.dur < time.Second {
						is.True(t, errors.Is(ctx.Err(), context.DeadlineExceeded))

						return
					}

					results := make(map[int]int, 4)
					results[runner1.id] = int(runner1.count.Load())
					results[runner2.id] = int(runner1.count.Load())
					results[runner3.id] = int(runner1.count.Load())
					results[0] = int(values.Load())

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
