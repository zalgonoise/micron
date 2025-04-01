package micron

import (
	"log/slog"

	"github.com/zalgonoise/cfg"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/executor"
	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/selector"
)

const (
	minBufferSize     = 64
	minAlloc          = 64
	defaultBufferSize = 1024
)

type Config struct {
	errBufferSize int

	sel   Selector
	execs []executor.Executor

	handler slog.Handler
	metrics Metrics
	tracer  trace.Tracer
}

func defaultConfig() *Config {
	return &Config{
		errBufferSize: minBufferSize,
		handler:       log.NoOp(),
		metrics:       metrics.NoOp(),
		tracer:        noop.NewTracerProvider().Tracer("no-op tracer"),
	}
}

// WithSelector configures the Runtime with the input selector.Selector.
//
// This call returns a cfg.NoOp cfg.Option if the input selector.Selector is nil, or if it is a
// selector.NoOp type.
func WithSelector(sel Selector) cfg.Option[*Config] {
	if sel == nil || sel == selector.NoOp() {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.sel = sel

		return config
	})
}

// WithJob adds a new executor.Executor to the Runtime configuration from the input ID, cron string and
// set of executor.Runner.
//
// This call returns a cfg.NoOp cfg.Option if no executor.Runner is provided, or if creating the executor.Executor
// fails (e.g. due to an invalid cron string).
//
// The gathered executor.Executor are then injected into a new selector.Selector that the Runtime will use.
//
// Note: this call is only valid if when creating a new Runtime via the New function, no WithSelector option is
// supplied; only WithJob. A call to New supports multiple WithJob cfg.Option.
func WithJob(id, cron string, runners ...executor.Runner) cfg.Option[*Config] {
	if len(runners) == 0 {
		return cfg.NoOp[*Config]{}
	}

	if id == "" {
		id = cron
	}

	exec, err := executor.New(id, runners,
		executor.WithSchedule(cron),
	)
	if err != nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		if config.execs == nil {
			config.execs = make([]executor.Executor, 0, minAlloc)
		}

		config.execs = append(config.execs, exec)

		return config
	})
}

// WithErrorBufferSize defines the capacity of the error channel that the Runtime exposes in
// its Runtime.Err method.
func WithErrorBufferSize(size int) cfg.Option[*Config] {
	if size < 0 {
		size = defaultBufferSize
	}

	return cfg.Register(func(config *Config) *Config {
		config.errBufferSize = size

		return config
	})
}

// WithMetrics decorates the Runtime with the input metrics registry.
func WithMetrics(m Metrics) cfg.Option[*Config] {
	if m == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.metrics = m

		return config
	})
}

// WithLogger decorates the Runtime with the input logger.
func WithLogger(logger *slog.Logger) cfg.Option[*Config] {
	if logger == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.handler = logger.Handler()

		return config
	})
}

// WithLogHandler decorates the Runtime with logging using the input log handler.
func WithLogHandler(handler slog.Handler) cfg.Option[*Config] {
	if handler == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.handler = handler

		return config
	})
}

// WithTrace decorates the Runtime with the input trace.Tracer.
func WithTrace(tracer trace.Tracer) cfg.Option[*Config] {
	if tracer == nil {
		return cfg.NoOp[*Config]{}
	}

	return cfg.Register(func(config *Config) *Config {
		config.tracer = tracer

		return config
	})
}
