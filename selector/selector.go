package selector

import (
	"context"
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/errs"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

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

// Metrics describes the actions that register Selector-related metrics.
type Metrics interface {
	// IncSelectorSelectCalls increases the count of Select calls, by the Selector.
	IncSelectorSelectCalls(ctx context.Context)
	// IncSelectorSelectErrors increases the count of Select call errors, by the Selector.
	IncSelectorSelectErrors(ctx context.Context)
}

type Selector struct {
	timeout time.Duration
	exec    []executor.Executor

	logger  *slog.Logger
	metrics Metrics
	tracer  trace.Tracer
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
func (s *Selector) Next(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "Selector.Select")
	defer span.End()

	s.metrics.IncSelectorSelectCalls(ctx)
	s.logger.InfoContext(ctx, "selecting the next task")

	// minStepDuration ensures that each execution is locked to the seconds mark and
	// a runner is not executed more than once per trigger.
	defer time.Sleep(minStepDuration)

	if len(s.exec) == 0 {
		err := ErrEmptyExecutorsList

		s.metrics.IncSelectorSelectCalls(ctx)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		s.logger.ErrorContext(ctx, "no tasks configured for execution",
			slog.String("error", err.Error()),
		)

		return err
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

			return
		default:
			errCh <- err
		}
	}()

	select {
	case <-localCtx.Done():
		return nil
	case err, ok := <-errCh:
		if !ok {
			return nil
		}

		if err == nil {
			return nil
		}

		s.metrics.IncSelectorSelectCalls(ctx)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		s.logger.ErrorContext(ctx, "failed to select and execute the next task",
			slog.String("error", err.Error()),
		)

		return err
	}
}

func (s *Selector) next(ctx context.Context) []executor.Executor {
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
func New(options ...cfg.Option[*Config]) (*Selector, error) {
	config := cfg.Set(defaultConfig(), options...)

	if len(config.exec) == 0 {
		return nil, ErrEmptyExecutorsList
	}

	if config.timeout < minStepDuration {
		config.timeout = defaultTimeout
	}

	return &Selector{
		timeout: config.timeout,
		exec:    config.exec,
		logger:  config.logger,
		metrics: config.metrics,
		tracer:  config.tracer,
	}, nil
}

func NewBlockingSelector(options ...cfg.Option[*Config]) (*BlockingSelector, error) {
	config := cfg.Set(defaultConfig(), options...)

	if len(config.exec) == 0 {
		return nil, ErrEmptyExecutorsList
	}

	return &BlockingSelector{
		exec:    config.exec,
		logger:  config.logger,
		metrics: config.metrics,
		tracer:  config.tracer,
	}, nil
}

// NoOp returns a no-op Selector.
func NoOp() noOpSelector {
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
