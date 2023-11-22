package selector

import (
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"go.opentelemetry.io/otel/trace"

	"github.com/zalgonoise/micron/executor"
)

type Config struct {
	exec    []executor.Executor
	block   bool
	timeout time.Duration

	handler slog.Handler
	metrics Metrics
	tracer  trace.Tracer
}

// WithExecutors configures the Selector with the input executor.Executor(s).
//
// This call returns a cfg.NoOp cfg.Option if the input set of executor.Executor is empty, or contains
// only nil and / or no-op executor.Executor.
func WithExecutors(executors ...executor.Executor) cfg.Option[Config] {
	execs := make([]executor.Executor, 0, len(executors))
	for i := range executors {
		if executors[i] == nil || executors[i] == executor.NoOp() {
			continue
		}

		execs = append(execs, executors[i])
	}

	if len(execs) == 0 {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		if len(config.exec) == 0 {
			config.exec = execs

			return config
		}

		config.exec = append(config.exec, execs...)

		return config
	})
}

// WithBlock configures the Selector to block (wait) for the underlying executor.Executor to complete the task.
//
// By default, the returned Selector from New is a non-blocking Selector. It mostly relies on the setup of the
// components to at least register the error as metrics or logs, but Exec calls return nil errors if the local context
// times out.
//
// WithBlock waits until the execution is done, so an accurate error value is returned from the Selector.
func WithBlock() cfg.Option[Config] {
	return cfg.Register(func(config Config) Config {
		config.block = true

		return config
	})
}

// WithTimeout configures a (non-blocking) Selector to wait a certain duration before detaching of the executable task,
// before continuing to select the next one.
//
// By default, the local context timeout is set to one second. Any negative or zero duration values result in a cfg.NoOp
// cfg.Option being returned.
func WithTimeout(dur time.Duration) cfg.Option[Config] {
	if dur <= 0 {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.timeout = dur

		return config
	})
}

// WithMetrics decorates the Selector with the input metrics registry.
func WithMetrics(m Metrics) cfg.Option[Config] {
	if m == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.metrics = m

		return config
	})
}

// WithLogger decorates the Selector with the input logger.
func WithLogger(logger *slog.Logger) cfg.Option[Config] {
	if logger == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.handler = logger.Handler()

		return config
	})
}

// WithLogHandler decorates the Selector with logging using the input log handler.
func WithLogHandler(handler slog.Handler) cfg.Option[Config] {
	if handler == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.handler = handler

		return config
	})
}

// WithTrace decorates the Selector with the input trace.Tracer.
func WithTrace(tracer trace.Tracer) cfg.Option[Config] {
	if tracer == nil {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.tracer = tracer

		return config
	})
}
