package executor

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/micron/v3/schedule"
	"github.com/zalgonoise/micron/v3/schedule/cronlex"
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

	is.Equal(t, 0, noOp.Next(context.Background(), time.Now()))
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

func TestIsReboot(t *testing.T) {
	for _, testcase := range []struct {
		name  string
		e     *Executable
		wants bool
	}{
		{
			name: "IsReboot",
			e: &Executable{
				cron: &schedule.RebootSchedule{},
			},
			wants: true,
		},
		{
			name: "IsNotReboot",
			e: &Executable{
				cron: &schedule.CronSchedule{},
			},
			wants: false,
		},
		{
			name:  "CronIsNil",
			e:     &Executable{},
			wants: false,
		},
		{
			name:  "ExecutableIsNil",
			wants: false,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			isReboot := IsReboot(testcase.e)
			is.Equal(t, testcase.wants, isReboot)
		})
	}
}

func TestJoinSchedules(t *testing.T) {
	everyMinute, err := cronlex.Parse("* * * * *")
	is.NilError(t, err)
	every2ndMinute, err := cronlex.Parse("*/2 * * * *")
	is.NilError(t, err)

	everyMinSchedule, err := schedule.New(schedule.WithSchedule(everyMinute), schedule.WithLocation(time.UTC))
	is.NilError(t, err)
	every2ndMinSchedule, err := schedule.New(schedule.WithSchedule(every2ndMinute), schedule.WithLocation(time.UTC))
	is.NilError(t, err)

	exec1 := &Executable{id: "1", cron: everyMinSchedule}
	exec2 := &Executable{id: "2", cron: everyMinSchedule}
	exec3 := &Executable{id: "3", cron: every2ndMinSchedule}

	for _, testcase := range []struct {
		name  string
		execs []*Executable
		wants map[*cronlex.Schedule][]*Executable
	}{
		{
			name:  "SameSchedule",
			execs: []*Executable{exec1, exec2},
			wants: map[*cronlex.Schedule][]*Executable{
				everyMinute: {exec1, exec2},
			},
		},
		{
			name:  "DifferentSchedules",
			execs: []*Executable{exec1, exec3},
			wants: map[*cronlex.Schedule][]*Executable{
				everyMinute:    {exec1},
				every2ndMinute: {exec3},
			},
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			m := JoinSchedules(testcase.execs)

			is.Equal(t, len(testcase.wants), len(m))
			for k, v := range testcase.wants {
				value, ok := m[k]
				is.True(t, ok)
				is.Equal(t, len(v), len(value))

				for i := range value {
					is.Equal(t, v[i], value[i])
				}
			}
		})
	}
}
