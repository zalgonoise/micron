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
	}

	for _, testcase := range []struct {
		name           string
		s              Selector
		handler        slog.Handler
		wants          Selector
		defaultHandler bool
	}{
		{
			name:  "NilSelector",
			wants: noOpSelector{},
		},
		{
			name:  "NoOpSelector",
			s:     noOpSelector{},
			wants: noOpSelector{},
		},
		{
			name: "NilHandler",
			s:    s,
			wants: withLogs{
				s: s,
			},
			defaultHandler: true,
		},
		{
			name:    "WithHandler",
			s:       s,
			handler: h,
			wants: withLogs{
				s:      s,
				logger: slog.New(h),
			},
		},
		{
			name:    "WithNoOpHandler",
			s:       s,
			handler: log.NoOp(),
			wants: withLogs{
				s: s,
			},
			defaultHandler: true,
		},
		{
			name: "ReplaceHandler",
			s: withLogs{
				s: s,
			},
			handler: h,
			wants: withLogs{
				s:      s,
				logger: slog.New(h),
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			s := AddLogs(testcase.s, testcase.handler)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			//nolint:errcheck // unit test with no-ops configured
			_ = s.Next(ctx)

			switch exec := s.(type) {
			case noOpSelector:
				is.Equal(t, testcase.wants, s)
			case withLogs:
				wants, ok := testcase.wants.(withLogs)
				is.True(t, ok)

				is.Equal(t, wants.s, exec.s)

				if testcase.defaultHandler {
					is.True(t, exec.logger.Handler() != nil)

					return
				}

				is.Equal(t, wants.logger.Handler(), exec.logger.Handler())
			}
		})
	}
}

type testMetrics struct{}

func (testMetrics) IncSelectorSelectCalls()  {}
func (testMetrics) IncSelectorSelectErrors() {}

type testExecutor struct{}

func (testExecutor) Exec(ctx context.Context) error     { return ctx.Err() }
func (testExecutor) Next(context.Context) (t time.Time) { return t }
func (testExecutor) ID() string                         { return "" }

type testSelector struct{}

func (testSelector) Next(ctx context.Context) error { return ctx.Err() }

func TestSelectorWithMetrics(t *testing.T) {
	m := testMetrics{}
	s := &selector{
		exec: []executor.Executor{testExecutor{}},
	}

	for _, testcase := range []struct {
		name  string
		s     Selector
		m     Metrics
		wants Selector
	}{
		{
			name:  "NilSelector",
			wants: noOpSelector{},
		},
		{
			name:  "NoOpSelector",
			s:     noOpSelector{},
			wants: noOpSelector{},
		},
		{
			name:  "NilMetrics",
			s:     s,
			wants: s,
		},
		{
			name:  "NoOpMetrics",
			s:     s,
			m:     metrics.NoOp(),
			wants: s,
		},
		{
			name: "WithMetrics",
			s:    s,
			m:    m,
			wants: withMetrics{
				s: s,
				m: m,
			},
		},
		{
			name: "ReplaceMetrics",
			s: withMetrics{
				s: s,
			},
			m: m,
			wants: withMetrics{
				s: s,
				m: m,
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			s := AddMetrics(testcase.s, testcase.m)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			//nolint:errcheck // unit test with no-ops configured
			_ = s.Next(ctx)

			switch sched := s.(type) {
			case noOpSelector:
				is.Equal(t, testcase.wants, s)
			case withMetrics:
				wants, ok := testcase.wants.(withMetrics)
				is.True(t, ok)
				is.Equal(t, wants.s, sched.s)
				is.Equal(t, wants.m, sched.m)
			}

			cancel()

			//nolint:errcheck // unit test with no-ops configured
			_ = s.Next(ctx)
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
	s := &selector{
		exec: []executor.Executor{executor.NoOp()},
	}

	for _, testcase := range []struct {
		name   string
		s      Selector
		tracer trace.Tracer
		wants  Selector
	}{
		{
			name:  "NilSelector",
			wants: noOpSelector{},
		},
		{
			name:  "NoOpSelector",
			s:     noOpSelector{},
			wants: noOpSelector{},
		},
		{
			name:  "NilTracer",
			s:     s,
			wants: s,
		},
		{
			name:   "WithTracer",
			s:      s,
			tracer: noop.NewTracerProvider().Tracer("test"),
			wants: withTrace{
				s:      s,
				tracer: noop.NewTracerProvider().Tracer("test"),
			},
		},
		{
			name: "ReplaceTracer",
			s: withTrace{
				s: s,
			},
			tracer: noop.NewTracerProvider().Tracer("test"),
			wants: withTrace{
				s:      s,
				tracer: noop.NewTracerProvider().Tracer("test"),
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			s := AddTraces(testcase.s, testcase.tracer)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			//nolint:errcheck // unit test with no-ops configured
			_ = s.Next(ctx)

			switch sched := s.(type) {
			case noOpSelector:
				is.Equal(t, testcase.wants, s)
			case withTrace:
				wants, ok := testcase.wants.(withTrace)
				is.True(t, ok)
				is.Equal(t, wants.s, sched.s)
				is.Equal(t, wants.tracer, sched.tracer)
			}
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
			blockingSelector{}.Next(context.Background()),
		))
	})
	t.Run("NonBlocking/FromRawSelector", func(t *testing.T) {
		is.True(t, errors.Is(
			ErrEmptyExecutorsList,
			selector{}.Next(context.Background()),
		))
	})

	t.Run("FromConstructor", func(t *testing.T) {
		_, err := New()
		is.True(t, errors.Is(ErrEmptyExecutorsList, err))
	})
}
