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
	}

	for _, testcase := range []struct {
		name           string
		e              Executor
		handler        slog.Handler
		wants          Executor
		defaultHandler bool
	}{
		{
			name:  "NilExecutor",
			wants: noOpExecutor{},
		},
		{
			name:  "NoOpExecutor",
			e:     noOpExecutor{},
			wants: noOpExecutor{},
		},
		{
			name: "NilHandler",
			e:    e,
			wants: withLogs{
				e: e,
			},
			defaultHandler: true,
		},
		{
			name:    "WithHandler",
			e:       e,
			handler: h,
			wants: withLogs{
				e:      e,
				logger: slog.New(h),
			},
		},
		{
			name:    "WithNoOpHandler",
			e:       e,
			handler: log.NoOp(),
			wants: withLogs{
				e: e,
			},
			defaultHandler: true,
		},
		{
			name: "ReplaceHandler",
			e: withLogs{
				e: e,
			},
			handler: h,
			wants: withLogs{
				e:      e,
				logger: slog.New(h),
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			e := AddLogs(testcase.e, testcase.handler)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_ = e.ID()
			_ = e.Next(ctx)

			//nolint:errcheck // unit test with no-ops configured
			_ = e.Exec(ctx)

			switch exec := e.(type) {
			case noOpExecutor, Executable:
				is.Equal(t, testcase.wants, e)
			case withLogs:
				wants, ok := testcase.wants.(withLogs)
				is.True(t, ok)

				is.Equal(t, wants.e, exec.e)

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

func (testMetrics) IncExecutorExecCalls(string)                               {}
func (testMetrics) IncExecutorExecErrors(string)                              {}
func (testMetrics) ObserveExecLatency(context.Context, string, time.Duration) {}
func (testMetrics) IncExecutorNextCalls(string)                               {}

func TestExecutorWithMetrics(t *testing.T) {
	m := testMetrics{}
	s, err := schedule.New(schedule.WithSchedule("* * * * * *"))
	is.Empty(t, err)

	e := &Executable{
		id:   "test",
		cron: s,
		runners: []Runner{Runnable(func(ctx context.Context) error {
			return ctx.Err()
		})},
	}

	for _, testcase := range []struct {
		name  string
		e     Executor
		m     Metrics
		wants Executor
	}{
		{
			name:  "NilExecutor",
			wants: noOpExecutor{},
		},
		{
			name:  "NoOpExecutor",
			e:     noOpExecutor{},
			wants: noOpExecutor{},
		},
		{
			name:  "NilMetrics",
			e:     e,
			wants: e,
		},
		{
			name: "WithMetrics",
			e:    e,
			m:    m,
			wants: withMetrics{
				e: e,
				m: m,
			},
		},
		{
			name:  "NoOpMetrics",
			e:     e,
			m:     metrics.NoOp(),
			wants: e,
		},
		{
			name: "ReplaceMetrics",
			e: withMetrics{
				e: e,
			},
			m: m,
			wants: withMetrics{
				e: e,
				m: m,
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			e := AddMetrics(testcase.e, testcase.m)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_ = e.ID()
			_ = e.Next(ctx)

			//nolint:errcheck // unit test with no-ops configured
			_ = e.Exec(ctx)

			switch sched := e.(type) {
			case noOpExecutor:
				is.Equal(t, testcase.wants, e)
			case withMetrics:
				wants, ok := testcase.wants.(withMetrics)
				is.True(t, ok)
				is.Equal(t, wants.e, sched.e)
				is.Equal(t, wants.m, sched.m)
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
	}

	for _, testcase := range []struct {
		name   string
		e      Executor
		tracer trace.Tracer
		wants  Executor
	}{
		{
			name:  "NilExecutor",
			wants: noOpExecutor{},
		},
		{
			name:  "NoOpExecutor",
			e:     noOpExecutor{},
			wants: noOpExecutor{},
		},
		{
			name:  "NilTracer",
			e:     e,
			wants: e,
		},
		{
			name:   "WithTracer",
			e:      e,
			tracer: noop.NewTracerProvider().Tracer("test"),
			wants: withTrace{
				e:      e,
				tracer: noop.NewTracerProvider().Tracer("test"),
			},
		},
		{
			name: "ReplaceTracer",
			e: withTrace{
				e: e,
			},
			tracer: noop.NewTracerProvider().Tracer("test"),
			wants: withTrace{
				e:      e,
				tracer: noop.NewTracerProvider().Tracer("test"),
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			e := AddTraces(testcase.e, testcase.tracer)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_ = e.ID()
			_ = e.Next(ctx)

			//nolint:errcheck // unit test with no-ops configured
			_ = e.Exec(ctx)

			switch sched := e.(type) {
			case noOpExecutor:
				is.Equal(t, testcase.wants, e)
			case withTrace:
				wants, ok := testcase.wants.(withTrace)
				is.True(t, ok)
				is.Equal(t, wants.e, sched.e)
				is.Equal(t, wants.tracer, sched.tracer)
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
