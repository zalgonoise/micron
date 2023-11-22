package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
)

const defaultPort = 13003

type Prometheus struct {
	server *http.Server

	schedulerNextCount       prometheus.Counter
	selectorSelectCount      prometheus.Counter
	selectorSelectErrorCount prometheus.Counter
	executorExecCount        *prometheus.CounterVec
	executorExecErrorCount   *prometheus.CounterVec
	executorLatency          *prometheus.HistogramVec
	executorNextCount        *prometheus.CounterVec
	cronUp                   prometheus.Gauge
}

func (m Prometheus) IncSchedulerNextCalls() {
	m.schedulerNextCount.Inc()
}

func (m Prometheus) IncSelectorSelectCalls() {
	m.selectorSelectCount.Inc()
}

func (m Prometheus) IncSelectorSelectErrors() {
	m.selectorSelectErrorCount.Inc()
}

func (m Prometheus) IncExecutorExecCalls(id string) {
	m.executorExecCount.WithLabelValues(id).Inc()
}

func (m Prometheus) IncExecutorExecErrors(id string) {
	m.executorExecErrorCount.WithLabelValues(id).Inc()
}

func (m Prometheus) ObserveExecLatency(ctx context.Context, id string, dur time.Duration) {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		m.executorLatency.
			WithLabelValues(id).(prometheus.ExemplarObserver).
			ObserveWithExemplar(
				dur.Seconds(),
				prometheus.Labels{traceIDKey: sc.TraceID().String()},
			)

		return
	}

	m.executorLatency.WithLabelValues(id).Observe(dur.Seconds())
}

func (m Prometheus) IncExecutorNextCalls(id string) {
	m.executorNextCount.WithLabelValues(id).Inc()
}

func (m Prometheus) IsUp(up bool) {
	if up {
		m.cronUp.Set(1.0)

		return
	}

	m.cronUp.Set(0.0)
}

func (m Prometheus) Registry() (*prometheus.Registry, error) {
	reg := prometheus.NewRegistry()

	for _, metric := range []prometheus.Collector{
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
			ReportErrors: false,
		}),
		m.schedulerNextCount,
		m.selectorSelectCount,
		m.selectorSelectErrorCount,
		m.executorExecCount,
		m.executorExecErrorCount,
		m.executorLatency,
		m.executorNextCount,
		m.cronUp,
	} {
		err := reg.Register(metric)
		if err != nil {
			return nil, err
		}
	}

	return reg, nil
}

func (m Prometheus) Shutdown(ctx context.Context) error {
	return m.server.Shutdown(ctx)
}

func newPrometheus(port int) (Metrics, error) {
	if port <= 0 {
		port = defaultPort
	}

	prom := Prometheus{
		schedulerNextCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "scheduler_next_calls_total",
			Help: "Count of time-calculations for the following scheduled task",
		}),
		selectorSelectCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "selector_select_calls_total",
			Help: "Count of selections done between multiple executors, for the next task",
		}),
		selectorSelectErrorCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "selector_select_errors_total",
			Help: "Count of errors when selecting the next task out of multiple executors",
		}),
		executorExecCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "executor_exec_calls_total",
			Help: "Count of executions from a single executor, identified by its ID",
		}, []string{"id"}),
		executorExecErrorCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "executor_exec_errors_total",
			Help: "Count of execution errors from a single executor, identified by its ID",
		}, []string{"id"}),
		executorLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "executor_exec_latency",
			Help:    "Histogram of execution times",
			Buckets: []float64{.00001, .00005, .0001, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}, []string{"id"}),
		executorNextCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "executor_exec_calls_total",
			Help: "Count of calls to retrieve the next execution time",
		}, []string{"id"}),
		cronUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "cron_up",
			Help: "Signals whether cron is running or not",
		}),
	}

	mux := http.NewServeMux()

	reg, err := prom.Registry()
	if err != nil {
		return noOpMetrics{}, err
	}

	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		Registry:          reg,
		EnableOpenMetrics: true,
	}))

	prom.server = &http.Server{
		Handler:      mux,
		Addr:         fmt.Sprintf(":%d", port),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		if err := prom.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	return prom, nil
}
