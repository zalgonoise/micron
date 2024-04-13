package micron

import "github.com/zalgonoise/micron/metrics"

// AddMetrics replaces the input Runtime's metrics, using the input Metrics interface.
//
// If the input metrics is nil, the Runtime's metrics will be set to be a no-op.
//
// If the input Runtime is nil or a no-op Runtime, a no-op Runtime is returned.
//
// If the input Runtime is a valid Runtime, then its metrics collector is replaced with the input one.
//
// Otherwise, the Runtime is returned as-is.
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
