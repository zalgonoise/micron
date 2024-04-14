package executor

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// AddTraces replaces the input Executor's tracer with a different one, using the input trace.Tracer.
//
// If the input tracer is nil, the Executor's tracer will be set to be a no-op.
//
// If the input Executor is nil or a no-op Executor, a no-op Executor is returned.
//
// If the input Executor is an Executable type, then its tracer is replaced with the input one.
//
// Otherwise, the Executor is returned as-is.
func AddTraces(e Executor, tracer trace.Tracer) Executor {
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("executor's no-op tracer")
	}

	if e == nil || e == NoOp() {
		return NoOp()
	}

	executable, ok := e.(*Executable)
	if !ok {
		return e
	}

	executable.tracer = tracer

	return executable
}
