package schedule

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/schedule/cronlex"
	"github.com/zalgonoise/micron/schedule/resolve"
)

func TestCronSchedule_Next(t *testing.T) {
	for _, testcase := range []struct {
		name  string
		cron  string
		sched Scheduler
		input time.Time
		wants time.Time
		err   error
	}{
		{
			name:  "Success/EverySecond",
			cron:  "* * * * * *",
			input: time.Date(2023, 10, 30, 10, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 30, 10, 12, 44, 0, time.UTC),
		},
		{
			name:  "Success/EveryFifthSecond",
			cron:  "*/5 * * * * *",
			input: time.Date(2023, 10, 30, 10, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 30, 10, 12, 45, 0, time.UTC),
		},
		{
			name:  "Success/EveryFifthSecondGoNext",
			cron:  "*/5 * * * * *",
			input: time.Date(2023, 10, 30, 10, 12, 45, 0, time.UTC),
			wants: time.Date(2023, 10, 30, 10, 12, 50, 0, time.UTC),
		},
		{
			name: "Success/SecondsOddCombo",
			cron: "0/3,2 * * * * *",

			input: time.Date(2023, 10, 30, 10, 12, 45, 0, time.UTC),
			wants: time.Date(2023, 10, 30, 10, 12, 48, 0, time.UTC),
		},
		{
			name:  "Success/EveryMinute",
			cron:  "* * * * *",
			input: time.Date(2023, 10, 30, 10, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 30, 10, 13, 0, 0, time.UTC),
		},
		{
			name:  "Success/OneHour",
			cron:  "0 * * * *",
			input: time.Date(2023, 10, 30, 10, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 30, 11, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/OneDay/WithDayChange",
			cron:  "0 0 * * *",
			input: time.Date(2023, 10, 30, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 31, 00, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithWeekday/NoWeekends",
			cron:  "0 0 * * 1-5",
			input: time.Date(2023, 10, 30, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 31, 00, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithWeekday/NoWeekendsAndWednesdays",
			cron:  "0 0 * * 1,2,4,5",
			input: time.Date(2023, 10, 31, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 11, 2, 00, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithRangesAndSteps/NoWeekendsAndWednesdays",
			cron:  "0 0 * * 1-2,4-5",
			input: time.Date(2023, 10, 31, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 11, 2, 00, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithRangesAndSteps/NoWeekendsAndWednesdays",
			cron:  "0 0/3,2 * * 1-2,4-5",
			input: time.Date(2023, 10, 31, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 11, 2, 00, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithWeekday/NoWeekendsStepSchedule",
			cron:  "0 0 * * 1,2,3,4,5",
			input: time.Date(2023, 10, 30, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 31, 00, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithStepSchedule/Every3Hours",
			cron:  "0 */3 * * *",
			input: time.Date(2023, 10, 30, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 31, 00, 0, 0, 0, time.UTC),
		},
		{
			name: "Success/EveryMinuteFromZeroToFive",
			cron: "0-5 * * * *",

			input: time.Date(2023, 10, 30, 10, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 30, 11, 0, 0, 0, time.UTC),
		},
		{
			name: "Success/InvalidCronString",
			cron: "*",

			input: time.Date(2023, 10, 30, 10, 12, 43, 0, time.UTC),
			err:   cronlex.ErrInvalidNodeType,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			sched, err := New(
				WithSchedule(testcase.cron),
				WithLocation(time.UTC),
				WithLogHandler(log.NoOp()),
				WithMetrics(metrics.NoOp()),
				WithTrace(noop.NewTracerProvider().Tracer("test")),
			)
			if testcase.err != nil {
				is.True(t, errors.Is(err, testcase.err))

				return
			}

			is.Empty(t, err)

			next := sched.Next(context.Background(), testcase.input)

			is.Equal(t, testcase.wants, next)
		})
	}
}

func TestConfig(t *testing.T) {
	t.Run("WithLogger", func(t *testing.T) {
		_, err := New(
			WithSchedule("* * * * *"),
			WithLocation(time.UTC),
			WithLogger(slog.New(log.NoOp())),
		)

		is.Empty(t, err)
	})

	t.Run("AllEmptyOptions", func(t *testing.T) {
		_, err := New(
			WithSchedule(""),
			WithLocation(nil),
			WithLogger(nil),
			WithLogHandler(nil),
			WithMetrics(nil),
			WithTrace(nil),
		)

		is.True(t, errors.Is(err, cronlex.ErrEmptyInput))
	})
}

func TestSchedulerWithLogs(t *testing.T) {
	h := slog.NewJSONHandler(io.Discard, nil)
	s := CronSchedule{
		Loc: time.Local,
		Schedule: cronlex.Schedule{
			Sec:      resolve.Everytime{},
			Min:      resolve.Everytime{},
			Hour:     resolve.Everytime{},
			DayMonth: resolve.Everytime{},
			Month:    resolve.Everytime{},
			DayWeek:  resolve.Everytime{},
		},
	}

	for _, testcase := range []struct {
		name           string
		s              Scheduler
		handler        slog.Handler
		wants          Scheduler
		defaultHandler bool
	}{
		{
			name:  "NilScheduler",
			wants: noOpScheduler{},
		},
		{
			name:  "NoOpScheduler",
			s:     noOpScheduler{},
			wants: noOpScheduler{},
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
			name:    "NoOpHandler",
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

			_ = s.Next(ctx, time.Time{})

			switch sched := s.(type) {
			case CronSchedule, noOpScheduler:
				is.Equal(t, testcase.wants, s)
			case withLogs:
				wants, ok := testcase.wants.(withLogs)
				is.True(t, ok)

				is.Equal(t, wants.s, sched.s)
				if testcase.defaultHandler {
					is.True(t, sched.logger.Handler() != nil)

					return
				}

				is.Equal(t, wants.logger.Handler(), sched.logger.Handler())
			}
		})
	}
}

type testMetrics struct{}

func (testMetrics) IncSchedulerNextCalls() {}

func TestSchedulerWithMetrics(t *testing.T) {
	m := testMetrics{}
	s := CronSchedule{
		Loc: time.Local,
		Schedule: cronlex.Schedule{
			Sec:      resolve.Everytime{},
			Min:      resolve.Everytime{},
			Hour:     resolve.Everytime{},
			DayMonth: resolve.Everytime{},
			Month:    resolve.Everytime{},
			DayWeek:  resolve.Everytime{},
		},
	}

	for _, testcase := range []struct {
		name  string
		s     Scheduler
		m     Metrics
		wants Scheduler
	}{
		{
			name:  "NilScheduler",
			wants: noOpScheduler{},
		},
		{
			name:  "NoOpScheduler",
			s:     noOpScheduler{},
			wants: noOpScheduler{},
		},
		{
			name:  "NilMetrics",
			s:     s,
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
			name:  "NoOpMetrics",
			s:     s,
			m:     metrics.NoOp(),
			wants: s,
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

			_ = s.Next(ctx, time.Time{})

			switch sched := s.(type) {
			case noOpScheduler:
				is.Equal(t, testcase.wants, s)
			case withMetrics:
				wants, ok := testcase.wants.(withMetrics)
				is.True(t, ok)
				is.Equal(t, wants.s, sched.s)
				is.Equal(t, wants.m, sched.m)
			}
		})
	}
}

func TestSchedulerWithTrace(t *testing.T) {
	s := CronSchedule{
		Loc: time.Local,
		Schedule: cronlex.Schedule{
			Sec:      resolve.Everytime{},
			Min:      resolve.Everytime{},
			Hour:     resolve.Everytime{},
			DayMonth: resolve.Everytime{},
			Month:    resolve.Everytime{},
			DayWeek:  resolve.Everytime{},
		},
	}

	for _, testcase := range []struct {
		name   string
		s      Scheduler
		tracer trace.Tracer
		wants  Scheduler
	}{
		{
			name:  "NilScheduler",
			wants: noOpScheduler{},
		},
		{
			name:  "NoOpScheduler",
			s:     noOpScheduler{},
			wants: noOpScheduler{},
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

			_ = s.Next(ctx, time.Time{})

			switch sched := s.(type) {
			case noOpScheduler:
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

	is.Equal(t, time.Time{}, noOp.Next(context.Background(), time.Now()))
}
