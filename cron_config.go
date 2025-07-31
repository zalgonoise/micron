package micron

import (
	"log/slog"

	"github.com/zalgonoise/cfg"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/v3/log"
	"github.com/zalgonoise/micron/v3/metrics"
	"github.com/zalgonoise/micron/v3/selector"
)

const (
	minBufferSize     = 64
	defaultBufferSize = 1024
)

func defaultRuntime() *Runtime {
	return &Runtime{
		err:     make(chan error, minBufferSize),
		logger:  slog.New(log.NoOp()),
		metrics: metrics.NoOp(),
		tracer:  noop.NewTracerProvider().Tracer("micron"),
	}
}

// WithSelector configures the Runtime with the input selector.Selector.
//
// This call returns a cfg.NoOp cfg.Option if the input selector.Selector is nil, or if it is a
// selector.NoOp type.
func WithSelector(sel Selector) cfg.Option[*Runtime] {
	if sel == nil || sel == selector.NoOp() {
		return cfg.NoOp[*Runtime]{}
	}

	return cfg.Register(func(r *Runtime) *Runtime {
		r.sel = sel

		return r
	})
}

// WithErrorBufferSize defines the capacity of the error channel that the Runtime exposes in
// its Runtime.Err method.
func WithErrorBufferSize(size int) cfg.Option[*Runtime] {
	if size < 0 {
		size = defaultBufferSize
	}

	return cfg.Register(func(r *Runtime) *Runtime {
		r.err = make(chan error, size)

		return r
	})
}

// WithMetrics decorates the Runtime with the input metrics registry.
func WithMetrics(m Metrics) cfg.Option[*Runtime] {
	if m == nil {
		return cfg.NoOp[*Runtime]{}
	}

	return cfg.Register(func(r *Runtime) *Runtime {
		r.metrics = m

		return r
	})
}

// WithLogger configures the Runtime with the input logger.
func WithLogger(logger *slog.Logger) cfg.Option[*Runtime] {
	if logger == nil {
		return cfg.NoOp[*Runtime]{}
	}

	return cfg.Register(func(r *Runtime) *Runtime {
		r.logger = logger

		return r
	})
}

// WithLogHandler configures the Runtime's logger using the input log handler.
func WithLogHandler(handler slog.Handler) cfg.Option[*Runtime] {
	if handler == nil {
		return cfg.NoOp[*Runtime]{}
	}

	return cfg.Register(func(r *Runtime) *Runtime {
		r.logger = slog.New(handler)

		return r
	})
}

// WithTrace configures the Runtime with the input trace.Tracer.
func WithTrace(tracer trace.Tracer) cfg.Option[*Runtime] {
	if tracer == nil {
		return cfg.NoOp[*Runtime]{}
	}

	return cfg.Register(func(r *Runtime) *Runtime {
		r.tracer = tracer

		return r
	})
}
