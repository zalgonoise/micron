package schedule

import (
	"github.com/zalgonoise/micron/metrics"
)

// AddMetrics replaces the input Scheduler's metrics, using the input Metrics interface.
//
// If the input metrics is nil, the Scheduler's metrics will be set to be a no-op.
//
// If the input Scheduler is nil or a no-op Scheduler, a no-op Scheduler is returned.
//
// If the input Scheduler is a valid Scheduler, then its metrics collector is replaced with the input one.
//
// Otherwise, the Scheduler is returned as-is.
func AddMetrics(s Scheduler, m Metrics) Scheduler {
	if m == nil || m == metrics.NoOp() {
		m = metrics.NoOp()
	}

	if s == nil || s == NoOp() {
		return NoOp()
	}

	sched, ok := s.(*CronSchedule)
	if !ok {
		return s
	}

	sched.metrics = m

	return sched
}
