package selector

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
)

func TestConfig(t *testing.T) {
	runner := executor.Runnable(func(context.Context) error {
		return nil
	})
	cron := "* * * * * *"

	exec, err := executor.New("test", []executor.Runner{runner},
		executor.WithSchedule(cron, time.Local),
	)
	is.Empty(t, err)

	for _, testcase := range []struct {
		name string
		opts []cfg.Option[*Config]
	}{
		{
			name: "WithExecutors/NoExecutors",
			opts: []cfg.Option[*Config]{
				WithExecutors(),
			},
		},
		{
			name: "WithExecutors/NilExecutor",
			opts: []cfg.Option[*Config]{
				WithExecutors(nil),
			},
		},
		{
			name: "WithExecutors/MultipleCalls",
			opts: []cfg.Option[*Config]{
				WithExecutors(exec),
				WithExecutors(exec),
			},
		},
		{
			name: "WithTimeout/Negative",
			opts: []cfg.Option[*Config]{
				WithTimeout(-3),
			},
		},
		{
			name: "WithTimeout/Zero",
			opts: []cfg.Option[*Config]{
				WithTimeout(0),
			},
		},
		{
			name: "WithTimeout/BelowMin",
			opts: []cfg.Option[*Config]{
				WithTimeout(30 * time.Millisecond),
			},
		},

		{
			name: "WithTimeout/OK",
			opts: []cfg.Option[*Config]{
				WithTimeout(100 * time.Millisecond),
			},
		},
		{
			name: "WithMetrics/NilMetrics",
			opts: []cfg.Option[*Config]{
				WithMetrics(nil),
			},
		},
		{
			name: "WithMetrics/NoOp",
			opts: []cfg.Option[*Config]{
				WithMetrics(metrics.NoOp()),
			},
		},
		{
			name: "WithLogger/NilLogger",
			opts: []cfg.Option[*Config]{
				WithLogger(nil),
			},
		},
		{
			name: "WithLogger/NoOp",
			opts: []cfg.Option[*Config]{
				WithLogger(slog.New(log.NoOp())),
			},
		},
		{
			name: "WithLogHandler/NilHandler",
			opts: []cfg.Option[*Config]{
				WithLogHandler(nil),
			},
		},
		{
			name: "WithLogHandler/NoOp",
			opts: []cfg.Option[*Config]{
				WithLogHandler(log.NoOp()),
			},
		},
		{
			name: "WithTrace/NilTracer",
			opts: []cfg.Option[*Config]{
				WithTrace(nil),
			},
		},
		{
			name: "WithTrace/NoOp",
			opts: []cfg.Option[*Config]{
				WithTrace(noop.NewTracerProvider().Tracer("test")),
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			_ = cfg.Set(new(Config), testcase.opts...)
		})
	}
}

func TestNoOp(t *testing.T) {
	noOp := NoOp()

	is.Empty(t, noOp.Next(context.Background()))
}

func TestWithObservability(t *testing.T) {
	runner := executor.Runnable(func(context.Context) error {
		return nil
	})

	testErr := errors.New("test error")
	errRunner := executor.Runnable(func(context.Context) error {
		return testErr
	})
	cron := "* * * * * *"

	for _, testcase := range []struct {
		name   string
		runner executor.Runner
		err    error
	}{
		{
			name:   "Success",
			runner: runner,
		},

		{
			name:   "WithError",
			runner: errRunner,
			err:    testErr,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			exec, err := executor.New("test", []executor.Runner{testcase.runner},
				executor.WithSchedule(cron, time.Local),
			)
			is.Empty(t, err)

			sel, err := New(
				WithExecutors(exec),
				WithLogHandler(log.NoOp()),
				WithLogger(slog.New(log.NoOp())),
				WithMetrics(metrics.NoOp()),
				WithTrace(noop.NewTracerProvider().Tracer("test")),
			)
			is.Empty(t, err)

			err = sel.Next(context.Background())
			is.True(t, errors.Is(err, testcase.err))
		})
	}
}

func TestZeroExecutors(t *testing.T) {
	t.Run("WithBlock/FromRawSelector", func(t *testing.T) {
		is.True(t, errors.Is(
			ErrEmptyExecutorsList,
			(&BlockingSelector{
				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}).Next(context.Background()),
		))
	})
	t.Run("NonBlocking/FromRawSelector", func(t *testing.T) {
		is.True(t, errors.Is(
			ErrEmptyExecutorsList,
			(&Selector{
				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}).Next(context.Background()),
		))
	})

	t.Run("FromConstructor", func(t *testing.T) {
		_, err := New()
		is.True(t, errors.Is(ErrEmptyExecutorsList, err))
	})
}
