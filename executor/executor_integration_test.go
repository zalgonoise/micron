////go:build integration

package executor_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/executor"
	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/schedule"
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

func testRunnable(ch chan<- int, value int) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		ch <- value

		return nil
	}
}

func TestNewExecutor(t *testing.T) {
	cron := "* * * * * *"
	sched, err := schedule.New(schedule.WithSchedule(cron))
	is.Empty(t, err)

	for _, testcase := range []struct {
		name    string
		id      string
		runners []executor.Runner
		opts    []cfg.Option[*executor.Config]
		err     error
	}{
		{
			name:    "DefaultID",
			runners: []executor.Runner{testRunner{}},
			opts: []cfg.Option[*executor.Config]{
				executor.WithSchedule(cron),
			},
		},
		{
			name:    "CustomScheduler",
			runners: []executor.Runner{testRunner{}},
			opts: []cfg.Option[*executor.Config]{
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
	values := make(chan int)
	runner1 := testRunner{v: 1, ch: values}
	runner2 := testRunner{v: 2, ch: values}
	runner3 := testRunner{v: 3, ch: values, err: testErr}
	runnable := testRunnable(values, 4)

	cron := "* * * * * *"
	defaultDur := 1010 * time.Millisecond

	for _, testcase := range []struct {
		name    string
		dur     time.Duration
		runners []executor.Runner
		wants   []int
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
			wants:   []int{1, 2},
		},
		{
			name:    "TwoRunnersAndARunnable",
			dur:     defaultDur,
			runners: []executor.Runner{runner1, runner2, executor.Runnable(runnable)},
			wants:   []int{1, 2, 4},
		},
		{
			name:    "NilRunnable",
			dur:     defaultDur,
			runners: []executor.Runner{executor.Runnable(nil)},
			wants:   []int{},
		},
		{
			name:    "ErrorRunner",
			dur:     defaultDur,
			runners: []executor.Runner{runner3},
			wants:   []int{3},
			err:     testErr,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			exec, err := executor.New(testcase.name, testcase.runners,
				executor.WithSchedule(cron),
				executor.WithLocation(time.Local),
				executor.WithMetrics(metrics.NoOp()),
				executor.WithLogHandler(log.NoOp()),
				executor.WithLogger(slog.New(log.NoOp())),
				executor.WithTrace(noop.NewTracerProvider().Tracer("test")),
			)
			is.Empty(t, err)
			is.Equal(t, testcase.name, exec.ID())

			results := make([]int, 0, len(testcase.wants))

			ctx, cancel := context.WithTimeout(context.Background(), testcase.dur)
			go func(t *testing.T, err error) {
				defer cancel()

				_ = exec.Next(ctx)

				execErr := exec.Exec(ctx)
				is.True(t, errors.Is(execErr, err))
			}(t, testcase.err)

			for {
				select {
				case <-ctx.Done():
					if testcase.dur < time.Second {
						is.True(t, errors.Is(ctx.Err(), context.DeadlineExceeded))

						return
					}

					is.EqualElements(t, testcase.wants, results)

					return
				case v := <-values:
					results = append(results, v)
				}
			}
		})
	}
}
