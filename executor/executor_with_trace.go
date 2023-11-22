package executor

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type withTrace struct {
	e      Executor
	tracer trace.Tracer
}

// Exec runs the task when on its scheduled time.
//
// For this, Exec leverages the Executor's underlying schedule.Scheduler to retrieve the job's next execution time,
// waits for it, and calls Runner.Run on each configured Runner. All raised errors are joined and returned at the end
// of this call.
func (e withTrace) Exec(ctx context.Context) error {
	ctx, span := e.tracer.Start(ctx, "Executor.Exec")
	defer span.End()

	span.SetAttributes(attribute.String("id", e.e.ID()))

	err := e.e.Exec(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

// Next calls the Executor's underlying schedule.Scheduler Next method.
func (e withTrace) Next(ctx context.Context) time.Time {
	ctx, span := e.tracer.Start(ctx, "Executor.Next")
	defer span.End()

	next := e.e.Next(ctx)

	span.SetAttributes(
		attribute.String("id", e.e.ID()),
		attribute.String("at", next.Format(time.RFC3339)),
	)

	return next
}

// ID returns this Executor's ID.
func (e withTrace) ID() string {
	return e.e.ID()
}

// AddTraces decorates the input Executor with tracing, using the input trace.Tracer.
//
// If the input Executor is nil or a no-op Executor, a no-op Executor is returned. If the input trace.Tracer is nil,
// then the input Executor is returned as-is.
//
// If the input Executor is already a Executor with tracing, then this Executor with tracing is returned with the new
// trace.Tracer configured in place of the former.
//
// Otherwise, the Executor is decorated with tracing within a custom type that implements Executor.
func AddTraces(e Executor, tracer trace.Tracer) Executor {
	if e == nil || e == NoOp() {
		return NoOp()
	}

	if tracer == nil {
		return e
	}

	if traced, ok := e.(withTrace); ok {
		traced.tracer = tracer

		return traced
	}

	return withTrace{
		e:      e,
		tracer: tracer,
	}
}
