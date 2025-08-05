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

	"github.com/zalgonoise/micron/v3/log"
	"github.com/zalgonoise/micron/v3/metrics"
)

type testScheduler struct{}

func (testScheduler) Next(context.Context, time.Time) time.Time { return time.Time{} }

func TestConfig(t *testing.T) {
	cron := "* * * * * *"

	for _, testcase := range []struct {
		name string
		opts []cfg.Option[*Executable]
	}{
		{
			name: "WithRunners/EmptyConfig",
			opts: []cfg.Option[*Executable]{},
		},
		{
			name: "WithScheduler/NoScheduler",
			opts: []cfg.Option[*Executable]{
				WithScheduler(nil),
			},
		},
		{
			name: "WithScheduler/OneScheduler",
			opts: []cfg.Option[*Executable]{
				WithScheduler(testScheduler{}),
			},
		},
		{
			name: "WithSchedule/EmptyString",
			opts: []cfg.Option[*Executable]{
				WithSchedule("", nil),
			},
		},
		{
			name: "WithSchedule/WithCronString",
			opts: []cfg.Option[*Executable]{
				WithSchedule(cron, time.Local),
			},
		},
		{
			name: "WithMetrics/NilMetrics",
			opts: []cfg.Option[*Executable]{
				WithMetrics(nil),
			},
		},
		{
			name: "WithMetrics/NoOp",
			opts: []cfg.Option[*Executable]{
				WithMetrics(metrics.NoOp()),
			},
		},
		{
			name: "WithLogger/NilLogger",
			opts: []cfg.Option[*Executable]{
				WithLogger(nil),
			},
		},
		{
			name: "WithLogger/NoOp",
			opts: []cfg.Option[*Executable]{
				WithLogger(slog.New(log.NoOp())),
			},
		},
		{
			name: "WithLogHandler/NilHandler",
			opts: []cfg.Option[*Executable]{
				WithLogHandler(nil),
			},
		},
		{
			name: "WithLogHandler/NoOp",
			opts: []cfg.Option[*Executable]{
				WithLogHandler(log.NoOp()),
			},
		},
		{
			name: "WithTrace/NilTracer",
			opts: []cfg.Option[*Executable]{
				WithTrace(nil),
			},
		},
		{
			name: "WithTrace/NoOp",
			opts: []cfg.Option[*Executable]{
				WithTrace(noop.NewTracerProvider().Tracer("test")),
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			_ = cfg.Set(new(Executable), testcase.opts...)
		})
	}
}

func TestNoOp(t *testing.T) {
	noOp := NoOp()

	is.Equal(t, time.Time{}, noOp.Next(context.Background(), time.Now()))
	is.Equal(t, "", noOp.ID())
	is.Empty(t, noOp.Exec(context.Background(), time.Now()))
}

func TestNew(t *testing.T) {
	cron := "@nope"
	r := Runnable(func(ctx context.Context) error {
		return nil
	})

	for _, testcase := range []struct {
		name    string
		runners []Runner
		conf    []cfg.Option[*Executable]
		err     error
	}{
		{
			name: "NoRunners",
			err:  ErrEmptyRunnerList,
		},
		{
			name:    "NoSchedulerOrCronString",
			runners: []Runner{r},
			conf:    []cfg.Option[*Executable]{},
			err:     ErrEmptyScheduler,
		},
		{
			name:    "InvalidCronString",
			runners: []Runner{r},
			conf: []cfg.Option[*Executable]{
				WithSchedule(cron, time.Local),
			},
			err: ErrEmptyScheduler,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			_, err := New(testcase.name, testcase.runners, testcase.conf...)
			is.True(t, errors.Is(err, testcase.err))
		})
	}
}
