package executor

import (
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/schedule"
	"github.com/zalgonoise/micron/schedule/cronlex"
)

func defaultExecutable() *Executable {
	return &Executable{
		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("micron.executor"),
	}
}

// WithScheduler configures the Executor with the input schedule.Scheduler.
//
// This call returns a cfg.NoOp cfg.Option if the input schedule.Scheduler is either nil or a no-op.
//
// Using this option does not require passing WithSchedule nor WithLocation options.
func WithScheduler(sched Scheduler) cfg.Option[*Executable] {
	if sched == nil || sched == schedule.NoOp() {
		return cfg.NoOp[*Executable]{}
	}

	return cfg.Register(func(e *Executable) *Executable {
		e.cron = sched

		return e
	})
}

// WithSchedule configures the Executor with a schedule.Scheduler using the input cron string.
//
// This call returns a cfg.NoOp cfg.Option if the cron string is empty.
//
// This option can be followed by a WithLocation option.
func WithSchedule(cron string, loc *time.Location) cfg.Option[*Executable] {
	if cron == "" {
		return cfg.NoOp[*Executable]{}
	}

	s, err := cronlex.Parse(cron)
	if err != nil {
		return cfg.NoOp[*Executable]{}
	}

	if loc == nil {
		loc = time.Local
	}

	scheduler, err := schedule.New(
		schedule.WithSchedule(s),
		schedule.WithLocation(loc),
	)

	if err != nil {
		return cfg.NoOp[*Executable]{}
	}

	return cfg.Register(func(e *Executable) *Executable {
		e.cron = scheduler

		return e
	})
}

// WithMetrics decorates the Executor with the input metrics registry.
func WithMetrics(m Metrics) cfg.Option[*Executable] {
	if m == nil {
		return cfg.NoOp[*Executable]{}
	}

	return cfg.Register(func(e *Executable) *Executable {
		e.metrics = m

		return e
	})
}

// WithLogger decorates the Executor with the input logger.
func WithLogger(logger *slog.Logger) cfg.Option[*Executable] {
	if logger == nil {
		return cfg.NoOp[*Executable]{}
	}

	return cfg.Register(func(e *Executable) *Executable {
		e.logger = logger

		return e
	})
}

// WithLogHandler decorates the Executor with logging using the input log handler.
func WithLogHandler(handler slog.Handler) cfg.Option[*Executable] {
	if handler == nil {
		return cfg.NoOp[*Executable]{}
	}

	return cfg.Register(func(e *Executable) *Executable {
		e.logger = slog.New(handler)

		return e
	})
}

// WithTrace decorates the Executor with the input trace.Tracer.
func WithTrace(tracer trace.Tracer) cfg.Option[*Executable] {
	if tracer == nil {
		return cfg.NoOp[*Executable]{}
	}

	return cfg.Register(func(config *Executable) *Executable {
		config.tracer = tracer

		return config
	})
}
