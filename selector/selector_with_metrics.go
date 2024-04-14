package selector

import (
	"github.com/zalgonoise/micron/metrics"
)

// AddMetrics replaces the input Selector's metrics, using the input Metrics interface.
//
// If the input metrics is nil, the Selector's metrics will be set to be a no-op.
//
// If the input Selector is nil or a no-op Selector, a no-op Selector is returned.
//
// If the input Selector is a valid Selector, then its metrics collector is replaced with the input one.
//
// Otherwise, the Selector is returned as-is.
func AddMetrics(s Selector, m Metrics) Selector {
	if m == nil || m == metrics.NoOp() {
		m = metrics.NoOp()
	}

	if s == nil || s == NoOp() {
		return NoOp()
	}

	switch sel := s.(type) {
	case selector:
		sel.metrics = m

		return sel
	case blockingSelector:
		sel.metrics = m

		return sel
	default:
		return s
	}
}
