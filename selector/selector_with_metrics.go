package selector

import (
	"context"

	"github.com/zalgonoise/micron/metrics"
)

// Metrics describes the actions that register Selector-related metrics.
type Metrics interface {
	// IncSelectorSelectCalls increases the count of Select calls, by the Selector.
	IncSelectorSelectCalls()
	// IncSelectorSelectErrors increases the count of Select call errors, by the Selector.
	IncSelectorSelectErrors()
}

type withMetrics struct {
	s Selector
	m Metrics
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
func (s withMetrics) Next(ctx context.Context) error {
	s.m.IncSelectorSelectCalls()

	if err := s.s.Next(ctx); err != nil {
		s.m.IncSelectorSelectErrors()

		return err
	}

	return nil
}

// AddMetrics decorates the input Selector with metrics, using the input Metrics interface.
//
// If the input Selector is nil or a no-op Selector, a no-op Selector is returned. If the input Metrics is nil or if it
// is a no-op Metrics interface, then the input Selector is returned as-is.
//
// If the input Selector is already a Selector with metrics, then this Selector with metrics is returned with the new
// Metrics interface configured in place of the former.
//
// Otherwise, the Selector is decorated with metrics within a custom type that implements Selector.
func AddMetrics(s Selector, m Metrics) Selector {
	if s == nil || s == NoOp() {
		return noOpSelector{}
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
