package executor

import (
	"github.com/zalgonoise/micron/metrics"
)

// AddMetrics replaces the input Executor's metrics, using the input Metrics interface.
//
// If the input metrics is nil, the Executor's metrics will be set to be a no-op.
//
// If the input Executor is nil or a no-op Executor, a no-op Executor is returned.
//
// If the input Executor is an Executable type, then its metrics collector is replaced with the input one.
//
// Otherwise, the Executor is returned as-is.
func AddMetrics(e Executor, m Metrics) Executor {
	if m == nil || m == metrics.NoOp() {
		m = metrics.NoOp()
	}

	if e == nil || e == NoOp() {
		return NoOp()
	}

	executable, ok := e.(*Executable)
	if !ok {
		return e
	}

	executable.metrics = m

	return executable
}
