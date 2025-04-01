package executor

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/schedule/cronlex"
)

type testScheduler struct{}

func (testScheduler) Next(context.Context, time.Time) time.Time { return time.Time{} }

func TestConfig(t *testing.T) {
	cron := "* * * * * *"

	for _, testcase := range []struct {
		name string
		opts []cfg.Option[*Config]
	}{
		{
			name: "WithRunners/EmptyConfig",
			opts: []cfg.Option[*Config]{},
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
		name    string
		runners []Runner
		conf    []cfg.Option[*Config]
		err     error
	}{
		{
			name: "NoRunners",
			err:  ErrEmptyRunnerList,
		},
		{
			name:    "NoSchedulerOrCronString",
			runners: []Runner{r},
			conf:    []cfg.Option[*Config]{},
			err:     ErrEmptyScheduler,
		},
		{
			name:    "InvalidCronString",
			runners: []Runner{r},
			conf: []cfg.Option[*Config]{
				WithSchedule(cron),
			},
			err: cronlex.ErrInvalidFrequency,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			_, err := New(testcase.name, testcase.runners, testcase.conf...)
			is.True(t, errors.Is(err, testcase.err))
		})
	}
}
