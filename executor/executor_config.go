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
)

type Config struct {
	scheduler Scheduler
	cron      string
	loc       *time.Location

	handler slog.Handler
	metrics Metrics
	tracer  trace.Tracer
}

func defaultConfig() *Config {
	return &Config{
		handler: log.NoOp(),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("executor's no-op tracer"),
	}
}

// WithScheduler configures the Executor with the input schedule.Scheduler.
//
// This call returns a cfg.NoOp cfg.Option if the input schedule.Scheduler is either nil or a no-op.
//
// Using this option does not require passing WithSchedule nor WithLocation options.
func WithScheduler(sched Scheduler) cfg.Option[*Config] {
	if sched == nil || sched == schedule.NoOp() {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.scheduler = sched

		return config
	})
}

// WithSchedule configures the Executor with a schedule.Scheduler using the input cron string.
//
// This call returns a cfg.NoOp cfg.Option if the cron string is empty.
//
// This option can be followed by a WithLocation option.
func WithSchedule(cron string) cfg.Option[*Config] {
	if cron == "" {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.cron = cron

		return config
	})
}

// WithLocation configures the Executor's schedule.Scheduler with the input time.Location.
//
// This call returns a cfg.NoOp cfg.Option if the input time.Location is nil.
//
// Using this option implies using the WithSchedule option, as it means the caller is creating a
// schedule from a cron string, instead of passing a schedule.Scheduler with the WithScheduler option.
func WithLocation(loc *time.Location) cfg.Option[*Config] {
	if loc == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.loc = loc

		return config
	})
}

// WithMetrics decorates the Executor with the input metrics registry.
func WithMetrics(m Metrics) cfg.Option[*Config] {
	if m == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.metrics = m

		return config
	})
}

// WithLogger decorates the Executor with the input logger.
func WithLogger(logger *slog.Logger) cfg.Option[*Config] {
	if logger == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.handler = logger.Handler()

		return config
	})
}

// WithLogHandler decorates the Executor with logging using the input log handler.
func WithLogHandler(handler slog.Handler) cfg.Option[*Config] {
	if handler == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.handler = handler

		return config
	})
}

// WithTrace decorates the Executor with the input trace.Tracer.
func WithTrace(tracer trace.Tracer) cfg.Option[*Config] {
	if tracer == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.tracer = tracer

		return config
	})
}
