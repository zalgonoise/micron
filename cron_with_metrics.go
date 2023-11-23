package micron

import (
	"context"

	"github.com/zalgonoise/micron/metrics"
)

// Metrics describes the actions that register Runtime-related metrics.
type Metrics interface {
	// IsUp signals whether the Runtime is running or not.
	IsUp(bool)
}

type withMetrics struct {
	r Runtime
	m Metrics
}

// Run kicks-off the cron module using the input context.Context.
//
// This is a blocking call that should be executed in a goroutine. The input context.Context can be leveraged to
// define when should the cron Runtime be halted, for example with context cancellation or timeout.
//
// Any error raised within a Run cycle is channeled to the Runtime errors channel, accessible with the Err method.
func (c withMetrics) Run(ctx context.Context) {
	c.m.IsUp(true)
	c.r.Run(ctx)
	c.m.IsUp(false)
}

// Err returns a receive-only errors channel, allowing the caller to consumer any errors raised during the execution
// of cron jobs.
//
// It is the responsibility of the caller to consume these errors appropriately, within the logic of their app.
func (c withMetrics) Err() <-chan error {
	return c.r.Err()
}

// AddMetrics decorates the input Runtime with metrics, using the input Metrics interface.
//
// If the input Runtime is nil or a no-op Runtime, a no-op Runtime is returned. If the input Metrics is nil or if it is
// a no-op Metrics interface, then the input Runtime is returned as-is.
//
// If the input Runtime is already a Runtime with metrics, then this Runtime with metrics is returned with the new
// Metrics interface configured in place of the former.
//
// Otherwise, the Runtime is decorated with metrics within a custom type that implements Runtime.
func AddMetrics(r Runtime, m Metrics) Runtime {
	if r == nil || r == NoOp() {
		return NoOp()
	}

	if m == nil || m == metrics.NoOp() {
		return r
	}

	if metric, ok := r.(withMetrics); ok {
		metric.m = m

		return metric
	}

	return withMetrics{
		r: r,
		m: m,
	}
}
