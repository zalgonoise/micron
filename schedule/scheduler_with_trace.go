package schedule

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type withTrace struct {
	s      Scheduler
	tracer trace.Tracer
}

// Next calculates and returns the following scheduled time, from the input time.Time.
func (s withTrace) Next(ctx context.Context, now time.Time) time.Time {
	ctx, span := s.tracer.Start(ctx, "Scheduler.Next")
	defer span.End()

	next := s.s.Next(ctx, now)

	span.SetAttributes(attribute.String("at", next.Format(time.RFC3339)))

	return next
}

// AddTraces decorates the input Scheduler with tracing, using the input trace.Tracer.
//
// If the input Scheduler is nil or a no-op Scheduler, a no-op Scheduler is returned. If the input trace.Tracer is nil,
// then the input Scheduler is returned as-is.
//
// If the input Scheduler is already a Scheduler with tracing, then this Scheduler with tracing is returned with the new
// trace.Tracer configured in place of the former.
//
// Otherwise, the Scheduler is decorated with tracing within a custom type that implements Scheduler.
func AddTraces(s Scheduler, tracer trace.Tracer) Scheduler {
	if s == nil || s == NoOp() {
		return NoOp()
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
