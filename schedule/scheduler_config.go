package schedule

import (
	"github.com/zalgonoise/micron/schedule/cronlex"
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
)

func defaultSchedule() *CronSchedule {
	return &CronSchedule{
		Loc:     time.Local,
		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("scheduler's no-op tracer"),
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

// WithMetrics decorates the Scheduler with the input metrics registry.
func WithMetrics(m Metrics) cfg.Option[*CronSchedule] {
	if m == nil {
		return cfg.NoOp[*CronSchedule]{}
	}

	return cfg.Register(func(s *CronSchedule) *CronSchedule {
		s.metrics = m

		return s
	})
}

// WithLogger decorates the Scheduler with the input logger.
func WithLogger(logger *slog.Logger) cfg.Option[*CronSchedule] {
	if logger == nil {
		return cfg.NoOp[*CronSchedule]{}
	}

	return cfg.Register(func(s *CronSchedule) *CronSchedule {
		s.logger = logger

		return s
	})
}

// WithLogHandler decorates the Scheduler with logging using the input log handler.
func WithLogHandler(handler slog.Handler) cfg.Option[*CronSchedule] {
	if handler == nil {
		return cfg.NoOp[*CronSchedule]{}
	}

	return cfg.Register(func(s *CronSchedule) *CronSchedule {
		s.logger = slog.New(handler)

		return s
	})
}

// WithTrace decorates the Scheduler with the input trace.Tracer.
func WithTrace(tracer trace.Tracer) cfg.Option[*CronSchedule] {
	if tracer == nil {
		return cfg.NoOp[*CronSchedule]{}
	}

	return cfg.Register(func(s *CronSchedule) *CronSchedule {
		s.tracer = tracer

		return s
	})
}
