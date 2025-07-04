package micron

import (
	"context"
	"log/slog"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/errs"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/zalgonoise/micron/log"
	"github.com/zalgonoise/micron/metrics"
	"github.com/zalgonoise/micron/selector"
)

const (
	errDomain = errs.Domain("micron")

	ErrEmpty = errs.Kind("empty")

	ErrSelector = errs.Entity("task selector")
)

var ErrEmptySelector = errs.WithDomain(errDomain, ErrEmpty, ErrSelector)

// Runtime describes the capabilities of a cron runtime, which allows a goroutine execution of its Run method,
// and has its errors channeled in the returned value from Err.
//
// Implementations of Runtime must focus on the uptime and closure of the cron component. For this, it will use
// the input context.Context in Run to allow being halted on-demand by the caller (with context cancellation or
// timeout).
//
// Considering that Run should be a blocking function to be executed in a goroutine, the Err method must expose an
// errors channel that pipes any raised errors during its execution back to the caller. It is the responsibility of the
// caller to consume these errors appropriately, within the logic of their app.
//
// A Runtime should leverage a selector.Selector to cycle through different jobs.
type Runtime interface {
	// Run kicks-off the cron module using the input context.Context.
	//
	// This is a blocking call that should be executed in a goroutine. The input context.Context can be leveraged to
	// define when should the cron Runtime be halted, for example with context cancellation or timeout.
	//
	// Any error raised within a Run cycle is channeled to the Runtime errors channel, accessible with the Err method.
	Run(ctx context.Context)
	// Err returns a receive-only errors channel, allowing the caller to consumer any errors raised during the execution
	// of cron jobs.
	//
	// It is the responsibility of the caller to consume these errors appropriately, within the logic of their app.
	Err() <-chan error
}

// Selector describes the capabilities of a cron selector, which picks up the next job to execute (out of a set of
// executor.Executor)
//
// Implementations of Selector must focus on the logic within its only method, Next, that will set the strategy to
// picking up the following job to run. The default implementation looks for the nearest job (in time) to execute, with
// support for multiple executions in one-go.
//
// Custom implementations could, for example, check for preconditions, run clean-up jobs, and more.
//
// The runtime of a Selector depends on the input context.Context when calling its Next method, as it can be used to
// signal cancellation or used for timeouts.
type Selector interface {
	// Next picks up the following scheduled job to execute from its configured (set of) executor.Executor, and
	// calls its Exec method.
	//
	// This call also imposes a minimum step duration of 50ms, to ensure that early-runs are not executed twice due to the
	// nature of using clocks in Go. This sleep is deferred to come in after the actual execution of the job.
	//
	// The Selector allows multiple executor.Executor to be configured, and multiple executor.Executor can share similar
	// execution times. If that is the case, the executor is launched in an executor.Multi call.
	//
	// The error returned from a Next call is the error raised by the executor.Executor's Exec call.
	Next(ctx context.Context) error
}

// Metrics describes the actions that register Runtime-related metrics.
type Metrics interface {
	// IsUp signals whether the Runtime is running or not.
	IsUp(bool)
}

type runtime struct {
	sel Selector

	err chan error

	logger  *slog.Logger
	metrics Metrics
	tracer  trace.Tracer
}

// Run kicks-off the cron module using the input context.Context.
//
// This is a blocking call that should be executed in a goroutine. The input context.Context can be leveraged to
// define when should the cron Runtime be halted, for example with context cancellation or timeout.
//
// Any error raised within a Run cycle is channeled to the Runtime errors channel, accessible with the Err method.
func (r runtime) Run(ctx context.Context) {
	ctx, span := r.tracer.Start(ctx, "Runtime.Run")
	defer span.End()

	r.logger.InfoContext(ctx, "starting cron")
	r.metrics.IsUp(true)

	defer func() {
		r.logger.InfoContext(ctx, "closing cron")
		r.metrics.IsUp(false)
		span.AddEvent("closing runtime")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := r.sel.Next(ctx); err != nil {
				r.err <- err
			}
		}
	}
}

// Err returns a receive-only error channel, allowing the caller to consumer any errors raised during the execution
// of cron jobs.
//
// It is the responsibility of the caller to consume these errors appropriately, within the logic of their app.
func (r runtime) Err() <-chan error {
	return r.err
}

// New creates a Runtime with the input cfg.Option, also returning an error if raised.
//
// The minimum requirements to create a Runtime is to supply either a selector.Selector through the WithSelector option,
// or a (set of) executor.Executor(s) through the WithJob option. The caller is free to select any they desire,
// and as such both means of creating this requirement are served as cfg.Option.
func New(options ...cfg.Option[*Config]) (Runtime, error) {
	config := cfg.Set(defaultConfig(), options...)

	return newRuntime(config)
}

func newRuntime(config *Config) (Runtime, error) {
	// validate input
	if config.sel == nil {
		if len(config.execs) == 0 {
			return NoOp(), errs.Join(ErrEmptySelector, selector.ErrEmptyExecutorsList)
		}

		sel, err := selector.New(selector.WithExecutors(config.execs...))
		if err != nil {
			return NoOp(), err
		}

		config.sel = sel
	}

	size := config.errBufferSize
	if size < minBufferSize {
		size = defaultBufferSize
	}

	r := runtime{
		sel: config.sel,
		err: make(chan error, size),

		logger:  slog.New(config.handler),
		metrics: config.metrics,
		tracer:  config.tracer,
	}

	if config.handler == nil {
		config.handler = log.NoOp()
	}

	r.logger = slog.New(config.handler)

	if config.metrics == nil {
		config.metrics = metrics.NoOp()
	}

	r.metrics = config.metrics

	if config.tracer == nil {
		config.tracer = noop.NewTracerProvider().Tracer("no-op tracer")
	}

	return r, nil
}

// NoOp returns a no-op Runtime.
func NoOp() Runtime {
	return noOpRuntime{}
}

type noOpRuntime struct{}

// Run kicks-off the cron module using the input context.Context.
//
// This is a no-op call and has no effect.
func (noOpRuntime) Run(context.Context) {}

// Err returns a receive-only errors channel, allowing the caller to consumer any errors raised during the execution
// of cron jobs.
//
// This is a no-op call and the returned receive-only errors channel is always nil.
func (noOpRuntime) Err() <-chan error {
	return nil
}
