package metrics

import (
	"context"
	"time"
)

func NoOp() Metrics {
	return noOpMetrics{}
}

type noOpMetrics struct{}

func (noOpMetrics) IncSchedulerNextCalls(context.Context)                     {}
func (noOpMetrics) IncSelectorSelectCalls(context.Context)                    {}
func (noOpMetrics) IncSelectorSelectErrors(context.Context)                   {}
func (noOpMetrics) IncExecutorExecCalls(context.Context, string)              {}
func (noOpMetrics) IncExecutorExecErrors(context.Context, string)             {}
func (noOpMetrics) ObserveExecLatency(context.Context, string, time.Duration) {}
func (noOpMetrics) IncExecutorNextCalls(context.Context, string)              {}
func (noOpMetrics) IsUp(context.Context, bool)                                {}
func (noOpMetrics) Shutdown(context.Context) error                            { return nil }
