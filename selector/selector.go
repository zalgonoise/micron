package selector

import (
	"context"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/errs"

	"github.com/zalgonoise/micron/executor"
)

const (
	minStepDuration = 50 * time.Millisecond
	defaultTimeout  = time.Second

	errSelectorDomain = errs.Domain("micron/selector")

	ErrEmpty = errs.Kind("empty")

	ErrExecutorsList = errs.Entity("executors list")
)

var ErrEmptyExecutorsList = errs.WithDomain(errSelectorDomain, ErrEmpty, ErrExecutorsList)

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

type selector struct {
	timeout time.Duration
	exec    []executor.Executor
}

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
func (s selector) Next(ctx context.Context) error {
	// minStepDuration ensures that each execution is locked to the seconds mark and
	// a runner is not executed more than once per trigger.
	defer time.Sleep(minStepDuration)

	if len(s.exec) == 0 {
		return ErrEmptyExecutorsList
	}

	localCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	errCh := make(chan error)

	go func() {
		var err error

		switch len(s.exec) {
		case 1:
			err = s.exec[0].Exec(ctx)
		default:
			err = executor.Multi(ctx, s.next(ctx)...)
		}

		select {
		case <-localCtx.Done():
			close(errCh)
		default:
			errCh <- err
		}
	}()

	select {
	case <-localCtx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func (s selector) next(ctx context.Context) []executor.Executor {
	var (
		next time.Duration
		exec = make([]executor.Executor, 0, len(s.exec))
		now  = time.Now()
	)

	for i := range s.exec {
		t := s.exec[i].Next(ctx).Sub(now)

		switch {
		case i == 0:
			next = t
			exec = append(exec, s.exec[i])

			continue
		case t == next:
			exec = append(exec, s.exec[i])

			continue
		case t < next:
			next = t
			exec = make([]executor.Executor, 0, len(s.exec))
			exec = append(exec, s.exec[i])

			continue
		}
	}

	return exec
}

// New creates a Selector with the input cfg.Option(s), also returning an error if raised.
//
// Creating a Selector requires at least one executor.Executor, which can be added through the WithExecutors option. To
// allow this configuration to be variadic as well, it is served as a cfg.Option.
func New(options ...cfg.Option[Config]) (Selector, error) {
	config := cfg.New(options...)

	sel, err := newSelector(config)
	if err != nil {
		return noOpSelector{}, err
	}

	if config.metrics != nil {
		sel = AddMetrics(sel, config.metrics)
	}

	if config.handler != nil {
		sel = AddLogs(sel, config.handler)
	}

	if config.tracer != nil {
		sel = AddTraces(sel, config.tracer)
	}

	return sel, nil
}

func newSelector(config Config) (Selector, error) {
	if len(config.exec) == 0 {
		return noOpSelector{}, ErrEmptyExecutorsList
	}

	if config.block {
		return blockingSelector{
			exec: config.exec,
		}, nil
	}

	if config.timeout < minStepDuration {
		config.timeout = defaultTimeout
	}

	return selector{
		timeout: config.timeout,
		exec:    config.exec,
	}, nil
}

// NoOp returns a no-op Selector
func NoOp() Selector {
	return noOpSelector{}
}

type noOpSelector struct{}

// Next picks up the following scheduled job to execute from its configured (set of) executor.Executor, and
// calls its Exec method.
//
// However, this is a no-op call, it has no effect and the returned error is always nil.
func (noOpSelector) Next(context.Context) error {
	return nil
}
