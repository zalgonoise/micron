package schedule

import (
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
)

type Config struct {
	cron string
	loc  *time.Location

	handler slog.Handler
	metrics Metrics
	tracer  trace.Tracer
}

func defaultConfig() Config {
	return Config{
		handler: log.NoOp(),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("scheduler's no-op tracer"),
	}
}

// WithSchedule configures the Scheduler with the input cron string.
//
// This call returns a cfg.NoOp cfg.Option if the input cron string is empty.
func WithSchedule(cron string) cfg.Option[Config] {
	if cron == "" {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.cron = cron

		return config
	})
}

// WithLocation configures the Scheduler with the input time.Location.
//
// This call returns a cfg.NoOp cfg.Option if the input time.Location is nil.
func WithLocation(loc *time.Location) cfg.Option[Config] {
	if loc == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.loc = loc

		return config
	})
}

// WithMetrics decorates the Scheduler with the input metrics registry.
func WithMetrics(m Metrics) cfg.Option[Config] {
	if m == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.metrics = m

		return config
	})
}

// WithLogger decorates the Scheduler with the input logger.
func WithLogger(logger *slog.Logger) cfg.Option[Config] {
	if logger == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.handler = logger.Handler()

		return config
	})
}

// WithLogHandler decorates the Scheduler with logging using the input log handler.
func WithLogHandler(handler slog.Handler) cfg.Option[Config] {
	if handler == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.handler = handler

		return config
	})
}

// WithTrace decorates the Scheduler with the input trace.Tracer.
func WithTrace(tracer trace.Tracer) cfg.Option[Config] {
	if tracer == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.tracer = tracer

		return config
	})
}
