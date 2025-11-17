package schedule

import (
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/v3/log"
	"github.com/zalgonoise/micron/v3/metrics"
	"github.com/zalgonoise/micron/v3/schedule/cronlex"
)

func defaultSchedule() *CronSchedule {
	return &CronSchedule{
		Loc:     time.UTC, // define UTC as standard default location
		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("micron.cron_schedule"),
	}
}

func defaultRebootSchedule() *RebootSchedule {
	return &RebootSchedule{
		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("micron.reboot_schedule"),
	}
}

// WithSchedule configures the Scheduler with the input cron string.
//
// This call returns a cfg.NoOp cfg.Option if the input cron string is empty.
func WithSchedule(cron *cronlex.Schedule) cfg.Option[*CronSchedule] {
	if cron == nil {
		return cfg.NoOp[*CronSchedule]{}
	}

	return cfg.Register(func(s *CronSchedule) *CronSchedule {
		s.Schedule = cron

		return s
	})
}

// WithLocation configures the Scheduler with the input time.Location.
//
// This call returns a cfg.NoOp cfg.Option if the input time.Location is nil.
func WithLocation(loc *time.Location) cfg.Option[*CronSchedule] {
	if loc == nil {
		loc = time.Local
	}

	return cfg.Register(func(s *CronSchedule) *CronSchedule {
		s.Loc = loc

		return s
	})
}

// WithObservability adds metrics, logs and / or tracing to the RebootSchedule.
func WithObservability(m Metrics, logger *slog.Logger, tracer trace.Tracer) cfg.Option[*CronSchedule] {
	if m == nil {
		m = metrics.NoOp()
	}

	if logger == nil {
		logger = slog.New(log.NoOp())
	}

	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("micron.cron_schedule")
	}

	return cfg.Register(func(s *CronSchedule) *CronSchedule {
		s.metrics = m
		s.logger = logger
		s.tracer = tracer

		return s
	})
}

// WithRebootObservability adds metrics, logs and / or tracing to the RebootSchedule.
func WithRebootObservability(m Metrics, logger *slog.Logger, tracer trace.Tracer) cfg.Option[*RebootSchedule] {
	if m == nil {
		m = metrics.NoOp()
	}

	if logger == nil {
		logger = slog.New(log.NoOp())
	}

	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("micron.reboot_schedule")
	}

	return cfg.Register(func(s *RebootSchedule) *RebootSchedule {
		s.metrics = m
		s.logger = logger
		s.tracer = tracer

		return s
	})
}
