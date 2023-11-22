package selector

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type withTrace struct {
	s      Selector
	tracer trace.Tracer
}

// Next picks up the following scheduled job to execute from its configured (set of) executor.Executor, and
// calls its Exec method.
//
// This call also imposes a minimum step duration of 50ms, to ensure that early-runs are not executed twice due to the
// nature of using clocks in Go. This sleep is deferred to come in after the actual execution of the job.
//
// The Selector allows multiple executor.Executor to be configured, and multiple executor.Executor can share similar
// execution times. If that is the case, the executor is launched in an executor.Multi call.
//
// The error returned from a Next call is the error raised by the executor.Executor's Exec call.
func (s withTrace) Next(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "Selector.Select")
	defer span.End()

	if err := s.s.Next(ctx); err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)

		return err
	}

	return nil
}

// AddTraces decorates the input Selector with tracing, using the input trace.Tracer.
//
// If the input Selector is nil or a no-op Selector, a no-op Selector is returned. If the input trace.Tracer is nil,
// then the input Selector is returned as-is.
//
// If the input Selector is already a Selector with tracing, then this Selector with tracing is returned with the new
// trace.Tracer configured in place of the former.
//
// Otherwise, the Selector is decorated with tracing within a custom type that implements Selector.
func AddTraces(s Selector, tracer trace.Tracer) Selector {
	if s == nil || s == NoOp() {
		return noOpSelector{}
	}

	if tracer == nil {
		return s
	}

	if traced, ok := s.(withTrace); ok {
		traced.tracer = tracer

		return traced
	}

	return withTrace{
		s:      s,
		tracer: tracer,
	}
}
