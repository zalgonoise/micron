package schedule

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/schedule/cronlex"
)

func TestCronSchedule_Next(t *testing.T) {
	for _, testcase := range []struct {
		name  string
		cron  string
		sched *CronSchedule
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
			wants: time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithWeekday/NoWeekends",
			cron:  "0 0 * * 1-5",
			input: time.Date(2023, 10, 30, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithWeekday/NoWeekendsAndWednesdays",
			cron:  "0 0 * * 1,2,4,5",
			input: time.Date(2023, 10, 31, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithRangesAndSteps/NoWeekendsAndWednesdays",
			cron:  "0 0 * * 1-2,4-5",
			input: time.Date(2023, 10, 31, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithRangesAndSteps/NoWeekendsAndWednesdays",
			cron:  "0 0/3,2 * * 1-2,4-5",
			input: time.Date(2023, 10, 31, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithWeekday/NoWeekendsStepSchedule",
			cron:  "0 0 * * 1,2,3,4,5",
			input: time.Date(2023, 10, 30, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "Success/WithStepSchedule/Every3Hours",
			cron:  "0 */3 * * *",
			input: time.Date(2023, 10, 30, 22, 12, 43, 0, time.UTC),
			wants: time.Date(2023, 10, 31, 0, 0, 0, 0, time.UTC),
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

func TestNoOp(t *testing.T) {
	noOp := NoOp()

	is.Equal(t, time.Time{}, noOp.Next(context.Background(), time.Now()))
}
