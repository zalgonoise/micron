package cron

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type withTrace struct {
	r      Runtime
	tracer trace.Tracer
}

// Run kicks-off the cron module using the input context.Context.
//
// This is a blocking call that should be executed in a goroutine. The input context.Context can be leveraged to
// define when should the cron Runtime be halted, for example with context cancellation or timeout.
//
// Any error raised within a Run cycle is channeled to the Runtime errors channel, accessible with the Err method.
func (c withTrace) Run(ctx context.Context) {
	ctx, span := c.tracer.Start(ctx, "Runtime.Run")
	defer span.End()

	c.r.Run(ctx)

	span.AddEvent("closing runtime")
}

// Err returns a receive-only errors channel, allowing the caller to consumer any errors raised during the execution
// of cron jobs.
//
// It is the responsibility of the caller to consume these errors appropriately, within the logic of their app.
func (c withTrace) Err() <-chan error {
	return c.r.Err()
}

// AddTraces decorates the input Runtime with tracing, using the input trace.Tracer.
//
// If the input Runtime is nil or a no-op Runtime, a no-op Runtime is returned. If the input trace.Tracer is nil, then
// the input Runtime is returned as-is.
//
// If the input Runtime is already a Runtime with tracing, then this Runtime with tracing is returned with the new
// trace.Tracer configured in place of the former.
//
// Otherwise, the Runtime is decorated with tracing within a custom type that implements Runtime.
func AddTraces(r Runtime, tracer trace.Tracer) Runtime {
	if r == nil || r == NoOp() {
		return NoOp()
	}

	if tracer == nil {
		return r
	}

	if traced, ok := r.(withTrace); ok {
		traced.tracer = tracer

		return traced
	}

	return withTrace{
		r:      r,
		tracer: tracer,
	}
}
