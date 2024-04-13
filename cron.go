package micron

import (
	"context"
	"log/slog"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/errs"
	"go.opentelemetry.io/otel/trace"

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

// Metrics describes the actions that register Runtime-related metrics.
type Metrics interface {
	// IsUp signals whether the Runtime is running or not.
	IsUp(bool)
}

type runtime struct {
	sel selector.Selector

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

// Err returns a receive-only errors channel, allowing the caller to consumer any errors raised during the execution
// of cron jobs.
//
// It is the responsibility of the caller to consume these errors appropriately, within the logic of their app.
func (r runtime) Err() <-chan error {
	return r.err
}

// New creates a Runtime with the input cfg.Option(s), also returning an error if raised.
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

	return runtime{
		sel: config.sel,
		err: make(chan error, size),

		logger:  slog.New(config.handler),
		metrics: config.metrics,
		tracer:  config.tracer,
	}, nil
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
