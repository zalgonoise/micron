package selector

import (
	"context"
	"iter"
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/errs"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	minStepDuration = 50 * time.Millisecond
	defaultTimeout  = time.Second

	errSelectorDomain = errs.Domain("micron/selector")

	ErrEmpty = errs.Kind("empty")

	ErrExecutorsList = errs.Entity("executors list")
)

var ErrEmptyExecutorsList = errs.WithDomain(errSelectorDomain, ErrEmpty, ErrExecutorsList)

// Executor describes the capabilities of cron job's executor component, which is based on fetching the next execution's
// time, Next; as well as running the job, Exec. It also exposes an ID method to allow access to this Executor's
// configured ID or name.
//
// Implementations of Executor must focus on the logic of the Exec method, which should contain the logic of the Next
// method as well. It should not be the responsibility of other components to wait until it is time to execute the job;
// but actually the Executor's responsibility to consider it in its Exec method. That being said, its Next method (just
// like its ID method) allows access to some of the details of the executor if the caller needs that information; as
// helpers.
//
// The logic behind Next and generally calculating the time for the next job execution should be deferred to a
// schedule.Scheduler, which should be part of the Executor.
//
// One Executor may contain multiple Runner, as a job may be composed of several (smaller) tasks. However, an Executor
// is identified by a single ID.
type Executor interface {
	// Exec runs the task when on its scheduled time.
	//
	// For this, Exec leverages the Executor's underlying schedule.Scheduler to retrieve the job's next execution time,
	// waits for it, and calls Runner.Run on each configured Runner. All raised errors are joined and returned at the end
	// of this call.
	Exec(ctx context.Context, now time.Time) error
	// Next calls the Executor's underlying schedule.Scheduler Next method.
	Next(ctx context.Context, now time.Time) time.Time
	// ID returns this Executor's ID.
	ID() string
}

type Clock interface {
	Now() time.Time
}

// Metrics describes the actions that register Selector-related metrics.
type Metrics interface {
	// IncSelectorSelectCalls increases the count of Select calls, by the Selector.
	IncSelectorSelectCalls(ctx context.Context)
	// IncSelectorSelectErrors increases the count of Select call errors, by the Selector.
	IncSelectorSelectErrors(ctx context.Context)
}

type Selector struct {
	timeout time.Duration
	exec    []Executor
	clock   Clock

	logger  *slog.Logger
	metrics Metrics
	tracer  trace.Tracer
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
		clock:   config.clock,
		logger:  config.logger,
		metrics: config.metrics,
		tracer:  config.tracer,
	}, nil
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

	localCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	errCh := make(chan error)
	now := s.clock.Now()

	switch len(s.exec) {
	case 0:
		s.metrics.IncSelectorSelectCalls(ctx)
		span.SetStatus(codes.Error, ErrEmptyExecutorsList.Error())
		span.RecordError(ErrEmptyExecutorsList)
		s.logger.ErrorContext(ctx, "no tasks configured for execution",
			slog.String("error", ErrEmptyExecutorsList.Error()),
		)

		close(errCh)

		return ErrEmptyExecutorsList
	case 1:
		go func() {
			if err := s.exec[0].Exec(ctx, now); err != nil {
				errCh <- err
			}

			close(errCh)
		}()
	default:
		s.next(ctx, now)(func(duration time.Duration, e Executor) bool {
			s.logger.DebugContext(ctx, "checking task to execute",
				slog.String("eta", duration.String()),
				slog.String("id", e.ID()),
			)

			if duration < minStepDuration {
				go func() {
					if err := e.Exec(ctx, now); err != nil {
						errCh <- err
					}
				}()
			}

			return true
		})

		close(errCh)
	}

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

func (s *Selector) next(ctx context.Context, now time.Time) iter.Seq2[time.Duration, Executor] {
	return func(yield func(dur time.Duration, exec Executor) bool) {
		for i := range s.exec {
			if !yield(s.exec[i].Next(ctx, now).Sub(now), s.exec[i]) {
				return
			}
		}
	}
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
