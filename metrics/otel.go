package metrics

import (
	"context"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

const defaultInterval = 500 * time.Millisecond
const ServiceName = "micron"

type ShutdownFunc func(ctx context.Context) error

func Meter() metric.Meter {
	return otel.GetMeterProvider().Meter(ServiceName)
}

var bucketBoundaries = []float64{
	.00001, .00005, .0001, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10,
}

type Otel struct {
	schedulerNextCount       metric.Int64Counter
	selectorSelectCount      metric.Int64Counter
	selectorSelectErrorCount metric.Int64Counter
	executorExecCount        metric.Int64Counter
	executorExecErrorCount   metric.Int64Counter
	executorLatency          metric.Float64Histogram
	executorNextCount        metric.Int64Counter
	cronUp                   metric.Int64Gauge
}

func NewOtel() (*Otel, error) {
	schedulerNextCount, err := Meter().Int64Counter(
		"scheduler_next_calls_total",
		metric.WithUnit("calls"),
		metric.WithDescription("Count of time-calculations for the following scheduled task"),
	)
	if err != nil {
		return nil, err
	}

	selectorSelectCount, err := Meter().Int64Counter(
		"selector_select_calls_total",
		metric.WithUnit("calls"),
		metric.WithDescription("Count of selections done between multiple executors, for the next task"),
	)
	if err != nil {
		return nil, err
	}

	selectorSelectErrorCount, err := Meter().Int64Counter(
		"selector_select_errors_total",
		metric.WithUnit("calls"),
		metric.WithDescription("Count of errors when selecting the next task out of multiple executors"),
	)
	if err != nil {
		return nil, err
	}

	executorExecCount, err := Meter().Int64Counter(
		"executor_exec_calls_total",
		metric.WithUnit("calls"),
		metric.WithDescription("Count of executions from a single executor, identified by its ID"),
	)
	if err != nil {
		return nil, err
	}

	executorExecErrorCount, err := Meter().Int64Counter(
		"executor_exec_errors_total",
		metric.WithUnit("calls"),
		metric.WithDescription("Count of execution errors from a single executor, identified by its ID"),
	)
	if err != nil {
		return nil, err
	}

	executorLatency, err := Meter().Float64Histogram(
		"executor_exec_latency",
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(bucketBoundaries...),
		metric.WithDescription("Histogram of execution times"),
	)
	if err != nil {
		return nil, err
	}

	executorNextCount, err := Meter().Int64Counter(
		"executor_exec_calls_total",
		metric.WithUnit("calls"),
		metric.WithDescription("Count of calls to retrieve the next execution time"),
	)
	if err != nil {
		return nil, err
	}

	cronUp, err := Meter().Int64Gauge(
		"cron_up",
		metric.WithUnit("up"),
		metric.WithDescription("Signals whether micron is running or not"),
	)
	if err != nil {
		return nil, err
	}

	return &Otel{
		schedulerNextCount:       schedulerNextCount,
		selectorSelectCount:      selectorSelectCount,
		selectorSelectErrorCount: selectorSelectErrorCount,
		executorExecCount:        executorExecCount,
		executorExecErrorCount:   executorExecErrorCount,
		executorLatency:          executorLatency,
		executorNextCount:        executorNextCount,
		cronUp:                   cronUp,
	}, nil
}

func (m *Otel) IncSchedulerNextCalls(ctx context.Context) {
	m.schedulerNextCount.Add(ctx, 1)
}

func (m *Otel) IncSelectorSelectCalls(ctx context.Context) {
	m.selectorSelectCount.Add(ctx, 1)
}

func (m *Otel) IncSelectorSelectErrors(ctx context.Context) {
	m.selectorSelectErrorCount.Add(ctx, 1)
}

func (m *Otel) IncExecutorExecCalls(ctx context.Context, id string) {
	m.executorExecCount.Add(ctx, 1, metric.WithAttributes(attribute.String("id", id)))
}

func (m *Otel) IncExecutorExecErrors(ctx context.Context, id string) {
	m.executorExecErrorCount.Add(ctx, 1, metric.WithAttributes(attribute.String("id", id)))
}

func (m *Otel) ObserveExecLatency(ctx context.Context, id string, dur time.Duration) {
	m.executorLatency.Record(ctx, dur.Seconds(), metric.WithAttributes(attribute.String("id", id)))
}

func (m *Otel) IncExecutorNextCalls(ctx context.Context, id string) {
	m.executorNextCount.Add(ctx, 1, metric.WithAttributes(attribute.String("id", id)))
}

func (m *Otel) IsUp(ctx context.Context, isUp bool) {
	var up int64
	if isUp {
		up = 1
	}

	m.cronUp.Record(ctx, up)
}

func Init(ctx context.Context, uri string) (ShutdownFunc, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(ServiceName)),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(uri),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithHeaders(map[string]string{
			"X-Scope-OrgID": "anonymous",
		}),
		otlpmetrichttp.WithRetry(otlpmetrichttp.RetryConfig{
			Enabled:         true,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     500 * time.Millisecond,
			MaxElapsedTime:  time.Minute,
		}),
	)
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(defaultInterval),
	)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}
