package metrics

import (
	"context"
	"time"

	"github.com/zalgonoise/cfg"
)

const (
	// traceIDKey is used as the trace ID key value in the prometheus.Labels in a prometheus.Exemplar.
	//
	// Its value of `trace_id` complies with the OpenTelemetry specification for metrics' exemplars, as seen in:
	// https://opentelemetry.io/docs/specs/otel/metrics/data-model/#exemplars
	traceIDKey = "trace_id"
)

type Metrics interface {
	IncSchedulerNextCalls(ctx context.Context)
	IncSelectorSelectCalls(ctx context.Context)
	IncSelectorSelectErrors(ctx context.Context)
	IncExecutorExecCalls(ctx context.Context, id string)
	IncExecutorExecErrors(ctx context.Context, id string)
	ObserveExecLatency(ctx context.Context, id string, dur time.Duration)
	IncExecutorNextCalls(ctx context.Context, id string)
	IsUp(ctx context.Context, isUp bool)
}

func New(options ...cfg.Option[Config]) (Metrics, error) {
	config := cfg.New(options...)

	switch config.metricsType {
	case metricsViaProm:
		return newPrometheus(config.serverPort)
	default:
		return newPrometheus(config.serverPort)
	}
}
