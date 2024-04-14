package selector

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace"
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

	exec, err := executor.New("test",
		executor.WithRunners(runner),
		executor.WithSchedule(cron),
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
			name: "WithBlock",
			opts: []cfg.Option[*Config]{
				WithBlock(),
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

func TestSelectorWithLogs(t *testing.T) {
	h := slog.NewJSONHandler(io.Discard, nil)
	s := &selector{
		exec: []executor.Executor{executor.NoOp()},

		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	for _, testcase := range []struct {
		name        string
		s           Selector
		handler     slog.Handler
		nilHandler  bool
		nilSelector bool
		noOpSel     bool
	}{
		{
			name:        "NilSelector",
			nilSelector: true,
		},
		{
			name:    "NoOpSelector",
			s:       noOpSelector{},
			noOpSel: true,
		},
		{
			name:       "NilHandler",
			s:          s,
			nilHandler: true,
		},
		{
			name:    "WithHandler",
			s:       s,
			handler: h,
		},
		{
			name:    "WithNoOpHandler",
			s:       s,
			handler: log.NoOp(),
		},
		{
			name: "ReplaceHandler",
			s: &blockingSelector{
				exec: []executor.Executor{executor.NoOp()},

				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			},
			handler: h,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			cronSelector := AddLogs(testcase.s, testcase.handler)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			//nolint:errcheck // unit test with no-ops configured
			_ = cronSelector.Next(ctx)

			switch sel := cronSelector.(type) {
			case noOpSelector:
				is.True(t, testcase.nilSelector || testcase.noOpSel)

				return
			case *selector:
				switch {
				case testcase.handler == nil:
					is.True(t, testcase.nilHandler)
				default:
					is.Equal(t, testcase.handler, sel.logger.Handler())
				}
			case *blockingSelector:
				switch {
				case testcase.handler == nil:
					is.True(t, testcase.nilHandler)
				default:
					is.Equal(t, testcase.handler, sel.logger.Handler())
				}
			}

			//nolint:errcheck // unit test with no-ops configured
			_ = cronSelector.Next(ctx)
		})
	}
}

type testSelector struct{}

func (testSelector) Next(ctx context.Context) error { return ctx.Err() }

func TestSelectorWithMetrics(t *testing.T) {
	m := metrics.NoOp()
	s := &selector{
		exec: []executor.Executor{executor.NoOp()},

		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	for _, testcase := range []struct {
		name        string
		s           Selector
		m           Metrics
		nilMetrics  bool
		nilSelector bool
		noOpSel     bool
	}{
		{
			name:        "NilSelector",
			nilSelector: true,
		},
		{
			name:    "NoOpSelector",
			s:       noOpSelector{},
			noOpSel: true,
		},
		{
			name:       "NilMetrics",
			s:          s,
			nilMetrics: true,
		},
		{
			name: "NoOpMetrics",
			s:    s,
			m:    metrics.NoOp(),
		},
		{
			name: "WithMetrics",
			s:    s,
			m:    m,
		},
		{
			name: "ReplaceMetrics",
			s: &blockingSelector{
				exec: []executor.Executor{executor.NoOp()},

				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			},
			m: m,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			cronSelector := AddMetrics(testcase.s, testcase.m)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			//nolint:errcheck // unit test with no-ops configured
			_ = cronSelector.Next(ctx)

			switch sel := cronSelector.(type) {
			case noOpSelector:
				is.True(t, testcase.nilSelector || testcase.noOpSel)

				return
			case *selector:
				switch {
				case testcase.m == nil:
					is.True(t, testcase.nilMetrics)
				default:
					is.Equal(t, testcase.m, sel.metrics)
				}
			case *blockingSelector:
				switch {
				case testcase.m == nil:
					is.True(t, testcase.nilMetrics)
				default:
					is.Equal(t, testcase.m, sel.metrics)
				}
			}

			//nolint:errcheck // unit test with no-ops configured
			_ = cronSelector.Next(ctx)
		})
	}

	t.Run("ErrorOnNext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		//nolint:errcheck // unit test with expected error
		_ = AddMetrics(testSelector{}, m).Next(ctx)
	})
}

func TestSelectorWithTrace(t *testing.T) {
	tracer := noop.NewTracerProvider().Tracer("test")
	s := &selector{
		exec: []executor.Executor{executor.NoOp()},

		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	for _, testcase := range []struct {
		name        string
		s           Selector
		tracer      trace.Tracer
		nilTracer   bool
		nilSelector bool
		noOpSel     bool
	}{
		{
			name:        "NilSelector",
			nilSelector: true,
		},
		{
			name:    "NoOpSelector",
			s:       noOpSelector{},
			noOpSel: true,
		},
		{
			name:      "NilTracer",
			s:         s,
			nilTracer: true,
		},
		{
			name:   "WithTracer",
			s:      s,
			tracer: tracer,
		},
		{
			name: "ReplaceTracer",
			s: &selector{
				exec: []executor.Executor{executor.NoOp()},

				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			},
			tracer: tracer,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			cronSelector := AddTraces(testcase.s, testcase.tracer)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			//nolint:errcheck // unit test with no-ops configured
			_ = cronSelector.Next(ctx)

			switch sel := cronSelector.(type) {
			case noOpSelector:
				is.True(t, testcase.nilSelector || testcase.noOpSel)

				return
			case *selector:
				switch {
				case testcase.tracer == nil:
					is.True(t, testcase.nilTracer)
				default:
					is.Equal(t, testcase.tracer, sel.tracer)
				}
			case *blockingSelector:
				switch {
				case testcase.tracer == nil:
					is.True(t, testcase.nilTracer)
				default:
					is.Equal(t, testcase.tracer, sel.tracer)
				}
			}

			//nolint:errcheck // unit test with no-ops configured
			_ = cronSelector.Next(ctx)
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
			exec, err := executor.New("test",
				executor.WithRunners(testcase.runner),
				executor.WithSchedule(cron),
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
			(&blockingSelector{
				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			}).Next(context.Background()),
		))
	})
	t.Run("NonBlocking/FromRawSelector", func(t *testing.T) {
		is.True(t, errors.Is(
			ErrEmptyExecutorsList,
			(&selector{
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
