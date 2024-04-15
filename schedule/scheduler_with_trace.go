package schedule

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// AddTraces replaces the input Scheduler's tracer with a different one, using the input trace.Tracer.
//
// If the input tracer is nil, the Scheduler's tracer will be set to be a no-op.
//
// If the input Scheduler is nil or a no-op Scheduler, a no-op Scheduler is returned.
//
// If the input Scheduler is a valid Scheduler, then its tracer is replaced with the input one.
//
// Otherwise, the Scheduler is returned as-is.
func AddTraces(s Scheduler, tracer trace.Tracer) Scheduler {
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("scheduler's no-op tracer")
	}

	if s == nil || s == NoOp() {
		return NoOp()
	}

	sched, ok := s.(*CronSchedule)
	if !ok {
		return s
	}

	sched.tracer = tracer

	return sched
}
