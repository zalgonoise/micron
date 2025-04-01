package micron

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/is"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/executor"
	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/selector"
)

func TestConfig(t *testing.T) {
	for _, testcase := range []struct {
		name string
		opts []cfg.Option[*Config]
	}{
		{
			name: "WithSelector/NilSelector",
			opts: []cfg.Option[*Config]{
				WithSelector(nil),
			},
		},
		{
			name: "WithSelector/NoOpSelector",
			opts: []cfg.Option[*Config]{
				WithSelector(selector.NoOp()),
			},
		},
		{
			name: "WithErrorBufferSize/Zero",
			opts: []cfg.Option[*Config]{
				WithErrorBufferSize(0),
			},
		},
		{
			name: "WithErrorBufferSize/Ten",
			opts: []cfg.Option[*Config]{
				WithErrorBufferSize(10),
			},
		},
		{
			name: "WithErrorBufferSize/Negative",
			opts: []cfg.Option[*Config]{
				WithErrorBufferSize(-10),
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

	noOp.Run(context.Background())
	is.Empty(t, noOp.Err())
}

func TestNew_NilSelector(t *testing.T) {
	_, err := New(nil)
	is.True(t, errors.Is(err, ErrEmptySelector))
}

func TestNewWithJob(t *testing.T) {
	runner1 := executor.Runnable(func(ctx context.Context) error {
		return nil
	})
	runner2 := executor.Runnable(func(ctx context.Context) error {
		return nil
	})

	type job struct {
		id      string
		cron    string
		runners []executor.Runner
	}

	for _, testcase := range []struct {
		name string
		jobs []job
		err  error
	}{
		{
			name: "Success/SingleRunner",
			jobs: []job{{
				id:      "ok-job",
				cron:    "* * * * * *",
				runners: []executor.Runner{runner1},
			}},
		},
		{
			name: "Success/MultiRunner",
			jobs: []job{{
				id:      "ok-job",
				cron:    "* * * * * *",
				runners: []executor.Runner{runner1, runner2},
			}},
		},
		{
			name: "Success/NoID/MultiRunner",
			jobs: []job{{
				cron:    "* * * * * *",
				runners: []executor.Runner{runner1, runner2},
			}},
		},
		{
			name: "Success/MultiJob/MultiRunner",
			jobs: []job{
				{
					id:      "seconds",
					cron:    "* * * * * *",
					runners: []executor.Runner{runner1},
				},
				{
					id:      "minutes",
					cron:    "* * * * *",
					runners: []executor.Runner{runner2},
				},
			},
		},
		{
			name: "Fail/NoCronString",
			jobs: []job{{
				id:      "ok-job",
				runners: []executor.Runner{runner1, runner2},
			}},
			err: ErrEmptySelector,
		},
		{
			name: "Fail/NoRunners",
			jobs: []job{{
				id:   "ok-job",
				cron: "* * * * * *",
			}},
			err: ErrEmptySelector,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			jobs := make([]cfg.Option[*Config], 0, len(testcase.jobs))

			for i := range testcase.jobs {
				jobs = append(jobs, WithJob(
					testcase.jobs[i].id,
					testcase.jobs[i].cron,
					testcase.jobs[i].runners...,
				))
			}

			_, err := New(jobs...)
			t.Log(err)
			is.True(t, errors.Is(err, testcase.err))
		})
	}
}
