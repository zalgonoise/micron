package micron

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// AddTraces replaces the input Runtime's tracer with a different one, using the input trace.Tracer.
//
// If the input tracer is nil, the Runtime's tracer will be set to be a no-op.
//
// If the input Runtime is nil or a no-op Runtime, a no-op Runtime is returned.
//
// If the input Runtime is a valid Runtime, then its tracer is replaced with the input one.
//
// Otherwise, the Runtime is returned as-is.
func AddTraces(r Runtime, tracer trace.Tracer) Runtime {
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("no-op cron runtime")
	}

	if r == nil || r == NoOp() {
		return NoOp()
	}

	cronRuntime, ok := r.(runtime)
	if !ok {
		return r
	}

	cronRuntime.tracer = tracer

	return cronRuntime
}
