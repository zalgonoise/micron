package micron

import "github.com/zalgonoise/micron/metrics"

// AddMetrics decorates the input Runtime with metrics, using the input Metrics interface.
//
// If the input metrics is nil, the Runtime's metrics will be set to be a no-op.
//
// If the input Runtime is nil or a no-op Runtime, a no-op Runtime is returned. If the input Metrics is nil or if it is
// a no-op Metrics interface, then the input Runtime is returned as-is.
//
// If the input Runtime is already a Runtime with metrics, then this Runtime with metrics is returned with the new
// Metrics interface configured in place of the former.
//
// Otherwise, the Runtime is decorated with metrics within a custom type that implements Runtime.
func AddMetrics(r Runtime, m Metrics) Runtime {
	if m == nil || m == metrics.NoOp() {
		m = metrics.NoOp()
	}

	if r == nil || r == NoOp() {
		return NoOp()
	}

	cronRuntime, ok := r.(runtime)
	if !ok {
		return r
	}

	cronRuntime.metrics = m

	return cronRuntime
}
