package executor

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

	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/schedule"
	"github.com/zalgonoise/micron/schedule/cronlex"
)

type testScheduler struct{}

func (testScheduler) Next(context.Context, time.Time) time.Time { return time.Time{} }

func TestConfig(t *testing.T) {
	runner := Runnable(func(context.Context) error {
		return nil
	})
	cron := "* * * * * *"

	for _, testcase := range []struct {
		name string
		opts []cfg.Option[*Config]
	}{
		{
			name: "WithRunners/NoRunners",
			opts: []cfg.Option[*Config]{
				WithRunners(),
			},
		},
		{
			name: "WithRunners/NilRunner",
			opts: []cfg.Option[*Config]{
				WithRunners(nil),
			},
		},
		{
			name: "WithRunners/OneRunner",
			opts: []cfg.Option[*Config]{
				WithRunners(runner),
			},
		},
		{
			name: "WithRunners/AddRunner",
			opts: []cfg.Option[*Config]{
				WithRunners(runner),
				WithRunners(runner),
			},
		},
		{
			name: "WithScheduler/NoScheduler",
			opts: []cfg.Option[*Config]{
				WithScheduler(nil),
			},
		},
		{
			name: "WithScheduler/OneScheduler",
			opts: []cfg.Option[*Config]{
				WithScheduler(testScheduler{}),
			},
		},
		{
			name: "WithSchedule/EmptyString",
			opts: []cfg.Option[*Config]{
				WithSchedule(""),
			},
		},
		{
			name: "WithSchedule/WithCronString",
			opts: []cfg.Option[*Config]{
				WithSchedule(cron),
			},
		},
		{
			name: "WithLocation/NilLocation",
			opts: []cfg.Option[*Config]{
				WithLocation(nil),
			},
		},
		{
			name: "WithLocation/Local",
			opts: []cfg.Option[*Config]{
				WithLocation(time.Local),
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

func TestExecutorWithLogs(t *testing.T) {
	h := slog.NewJSONHandler(io.Discard, nil)
	s, err := schedule.New(schedule.WithSchedule("* * * * * *"))
	is.Empty(t, err)

	e := &Executable{
		id:   "test",
		cron: s,
		runners: []Runner{Runnable(func(ctx context.Context) error {
			return ctx.Err()
		})},

		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	for _, testcase := range []struct {
		name        string
		e           Executor
		handler     slog.Handler
		nilHandler  bool
		nilExecutor bool
		noOpExec    bool
	}{
		{
			name:        "NilExecutor",
			nilExecutor: true,
		},
		{
			name:     "NoOpExecutor",
			e:        noOpExecutor{},
			noOpExec: true,
		},
		{
			name:       "NilHandler",
			e:          e,
			nilHandler: true,
		},
		{
			name:    "WithHandler",
			e:       e,
			handler: h,
		},
		{
			name:    "WithNoOpHandler",
			e:       e,
			handler: log.NoOp(),
		},
		{
			name: "ReplaceHandler",
			e: &Executable{
				id:   "test",
				cron: s,
				runners: []Runner{Runnable(func(ctx context.Context) error {
					return ctx.Err()
				})},

				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			},
			handler: h,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			exec := AddLogs(testcase.e, testcase.handler)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_ = exec.ID()
			_ = exec.Next(ctx)

			//nolint:errcheck // unit test with no-ops configured
			_ = exec.Exec(ctx)

			switch e := exec.(type) {
			case noOpExecutor:
				is.True(t, testcase.nilExecutor || testcase.noOpExec)
			case *Executable:
				switch {
				case testcase.handler == nil:
					is.True(t, testcase.nilHandler)
				default:
					is.Equal(t, testcase.handler, e.logger.Handler())
				}
			}
		})
	}
}

func TestExecutorWithMetrics(t *testing.T) {
	m := metrics.NoOp()
	s, err := schedule.New(schedule.WithSchedule("* * * * * *"))
	is.Empty(t, err)

	e := &Executable{
		id:   "test",
		cron: s,
		runners: []Runner{Runnable(func(ctx context.Context) error {
			return ctx.Err()
		})},

		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	for _, testcase := range []struct {
		name        string
		e           Executor
		m           Metrics
		nilMetrics  bool
		nilExecutor bool
		noOpExec    bool
	}{
		{
			name:        "NilExecutor",
			nilExecutor: true,
		},
		{
			name:     "NoOpExecutor",
			e:        noOpExecutor{},
			noOpExec: true,
		},
		{
			name:       "NilMetrics",
			e:          e,
			nilMetrics: true,
		},
		{
			name: "WithMetrics",
			e:    e,
			m:    m,
		},
		{
			name: "NoOpMetrics",
			e:    e,
			m:    metrics.NoOp(),
		},
		{
			name: "ReplaceMetrics",
			e: &Executable{
				id:   "test",
				cron: s,
				runners: []Runner{Runnable(func(ctx context.Context) error {
					return ctx.Err()
				})},

				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			},
			m: m,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			exec := AddMetrics(testcase.e, testcase.m)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_ = exec.ID()
			_ = exec.Next(ctx)

			//nolint:errcheck // unit test with no-ops configured
			_ = exec.Exec(ctx)

			switch e := exec.(type) {
			case noOpExecutor:
				is.True(t, testcase.nilExecutor || testcase.noOpExec)
			case *Executable:
				switch {
				case testcase.m == nil:
					is.True(t, testcase.nilMetrics)
				default:
					is.Equal(t, testcase.m, e.metrics)
				}
			}
		})
	}
}

func TestExecutorWithTrace(t *testing.T) {
	s, err := schedule.New(schedule.WithSchedule("* * * * * *"))
	is.Empty(t, err)

	e := &Executable{
		id:   "test",
		cron: s,
		runners: []Runner{Runnable(func(ctx context.Context) error {
			return ctx.Err()
		})},

		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}

	for _, testcase := range []struct {
		name       string
		e          Executor
		tracer     trace.Tracer
		nilTracer  bool
		nilRuntime bool
		noOpCron   bool
	}{
		{
			name:       "NilExecutor",
			nilRuntime: true,
		},
		{
			name:     "NoOpExecutor",
			e:        noOpExecutor{},
			noOpCron: true,
		},
		{
			name:      "NilTracer",
			e:         e,
			nilTracer: true,
		},
		{
			name:   "WithTracer",
			e:      e,
			tracer: noop.NewTracerProvider().Tracer("test"),
		},
		{
			name: "ReplaceTracer",
			e: &Executable{
				id:   "test",
				cron: s,
				runners: []Runner{Runnable(func(ctx context.Context) error {
					return ctx.Err()
				})},

				logger:  slog.New(log.NoOp()),
				metrics: metrics.NoOp(),
				tracer:  noop.NewTracerProvider().Tracer("test"),
			},
			tracer: noop.NewTracerProvider().Tracer("test"),
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			exec := AddTraces(testcase.e, testcase.tracer)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_ = exec.ID()
			_ = exec.Next(ctx)

			//nolint:errcheck // unit test with no-ops configured
			_ = exec.Exec(ctx)

			switch e := exec.(type) {
			case noOpExecutor:
				is.True(t, testcase.nilRuntime || testcase.noOpCron)
			case *Executable:
				switch {
				case testcase.tracer == nil:
					is.True(t, testcase.nilTracer)
				default:
					is.Equal(t, testcase.tracer, e.tracer)
				}
			}
		})
	}
}

func TestNoOp(t *testing.T) {
	noOp := NoOp()

	is.Equal(t, time.Time{}, noOp.Next(context.Background()))
	is.Equal(t, "", noOp.ID())
	is.Empty(t, noOp.Exec(context.Background()))
}

func TestNew(t *testing.T) {
	cron := "@nope"
	r := Runnable(func(ctx context.Context) error {
		return nil
	})

	for _, testcase := range []struct {
		name string
		conf []cfg.Option[*Config]
		err  error
	}{
		{
			name: "NoRunners",
			err:  ErrEmptyRunnerList,
		},
		{
			name: "NoSchedulerOrCronString",
			conf: []cfg.Option[*Config]{
				WithRunners(r),
			},
			err: ErrEmptyScheduler,
		},
		{
			name: "InvalidCronString",
			conf: []cfg.Option[*Config]{
				WithRunners(r),
				WithSchedule(cron),
			},
			err: cronlex.ErrInvalidFrequency,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			_, err := New(testcase.name, testcase.conf...)
			is.True(t, errors.Is(err, testcase.err))
		})
	}
}
