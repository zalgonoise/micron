package executor

import (
	"context"
	"time"

	"github.com/zalgonoise/micron/metrics"
)

// Metrics describes the actions that register Executor-related metrics.
type Metrics interface {
	// IncExecutorExecCalls increases the count of Exec calls, by the Executor.
	IncExecutorExecCalls(id string)
	// IncExecutorExecErrors increases the count of Exec call errors, by the Executor.
	IncExecutorExecErrors(id string)
	// ObserveExecLatency registers the duration of an Exec call, by the Executor.
	ObserveExecLatency(ctx context.Context, id string, dur time.Duration)
	// IncExecutorNextCalls increases the count of Next calls, by the Executor.
	IncExecutorNextCalls(id string)
}

type withMetrics struct {
	e Executor
	m Metrics
}

// Exec runs the task when on its scheduled time.
//
// For this, Exec leverages the Executor's underlying schedule.Scheduler to retrieve the job's next execution time,
// waits for it, and calls Runner.Run on each configured Runner. All raised errors are joined and returned at the end
// of this call.
func (e withMetrics) Exec(ctx context.Context) error {
	id := e.e.ID()
	e.m.IncExecutorExecCalls(id)

	before := time.Now()

	err := e.e.Exec(ctx)

	e.m.ObserveExecLatency(ctx, id, time.Since(before))

	if err != nil {
		e.m.IncExecutorExecErrors(id)
	}

	return err
}

// Next calls the Executor's underlying schedule.Scheduler Next method.
func (e withMetrics) Next(ctx context.Context) time.Time {
	e.m.IncExecutorNextCalls(e.e.ID())

	return e.e.Next(ctx)
}

// ID returns this Executor's ID.
func (e withMetrics) ID() string {
	return e.e.ID()
}

// AddMetrics decorates the input Executor with metrics, using the input Metrics interface.
//
// If the input Executor is nil or a no-op Executor, a no-op Executor is returned. If the input Metrics is nil or if it
// is a no-op Metrics interface, then the input Executor is returned as-is.
//
// If the input Executor is already a Executor with metrics, then this Executor with metrics is returned with the new
// Metrics interface configured in place of the former.
//
// Otherwise, the Executor is decorated with metrics within a custom type that implements Executor.
func AddMetrics(e Executor, m Metrics) Executor {
	if e == nil || e == NoOp() {
		return NoOp()
	}

	if m == nil || m == metrics.NoOp() {
		return e
	}

	if metric, ok := e.(withMetrics); ok {
		metric.m = m

		return metric
	}

	return withMetrics{
		e: e,
		m: m,
	}
}
