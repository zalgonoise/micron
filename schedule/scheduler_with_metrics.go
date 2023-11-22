package schedule

import (
	"context"
	"time"

	"github.com/zalgonoise/micron/metrics"
)

// Metrics describes the actions that register Scheduler-related metrics.
type Metrics interface {
	// IncSchedulerNextCalls increases the count of Next calls, by the Scheduler.
	IncSchedulerNextCalls()
}

type withMetrics struct {
	s Scheduler
	m Metrics
}

// Next calculates and returns the following scheduled time, from the input time.Time.
func (s withMetrics) Next(ctx context.Context, now time.Time) time.Time {
	s.m.IncSchedulerNextCalls()

	return s.s.Next(ctx, now)
}

// AddMetrics decorates the input Scheduler with metrics, using the input Metrics interface.
//
// If the input Scheduler is nil or a no-op Scheduler, a no-op Scheduler is returned. If the input Metrics is nil or if
// it is a no-op Metrics interface, then the input Scheduler is returned as-is.
//
// If the input Scheduler is already a Scheduler with metrics, then this Scheduler with metrics is returned with the new
// Metrics interface configured in place of the former.
//
// Otherwise, the Scheduler is decorated with metrics within a custom type that implements Scheduler.
func AddMetrics(s Scheduler, m Metrics) Scheduler {
	if s == nil || s == NoOp() {
		return NoOp()
	}

	if m == nil || m == metrics.NoOp() {
		return s
	}

	if metric, ok := s.(withMetrics); ok {
		metric.m = m

		return metric
	}

	return withMetrics{
		s: s,
		m: m,
	}
}
