package metrics

import (
	"context"
	"time"
)

func NoOp() Metrics {
	return noOpMetrics{}
}

type noOpMetrics struct{}

func (noOpMetrics) IncSchedulerNextCalls()                                    {}
func (noOpMetrics) IncSelectorSelectCalls()                                   {}
func (noOpMetrics) IncSelectorSelectErrors()                                  {}
func (noOpMetrics) IncExecutorExecCalls(string)                               {}
func (noOpMetrics) IncExecutorExecErrors(string)                              {}
func (noOpMetrics) ObserveExecLatency(context.Context, string, time.Duration) {}
func (noOpMetrics) IncExecutorNextCalls(string)                               {}
func (noOpMetrics) IsUp(bool)                                                 {}
func (noOpMetrics) Shutdown(context.Context) error                            { return nil }
