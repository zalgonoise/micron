package selector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/micron/v3/executor"
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
	Next(ctx context.Context, now time.Time) time.Duration
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
	exec    []*executor.Executable
	errChan chan error
	err     *atomic.Pointer[error]
	cancel  context.CancelFunc
	clock   Clock

	logger  *slog.Logger
	metrics Metrics
	tracer  trace.Tracer
}

// New creates a Selector with the input cfg.Option(s), also returning an error if raised.
//
// Creating a Selector requires at least one executor.Executor, which can be added through the WithExecutors option. To
// allow this configuration to be variadic as well, it is served as a cfg.Option.
func New(ctx context.Context, options ...cfg.Option[*Config]) (*Selector, error) {
	config := cfg.Set(defaultConfig(), options...)

	if len(config.exec) == 0 {
		return nil, ErrEmptyExecutorsList
	}

	if config.timeout < minStepDuration {
		config.timeout = defaultTimeout
	}

	errChan := make(chan error)
	execs, rebootExecs := JoinSchedules(config.exec)

	ctx, cancel := context.WithCancel(ctx)

	s := &Selector{
		timeout: config.timeout,
		exec:    execs,
		errChan: errChan,
		err:     &atomic.Pointer[error]{},
		cancel:  cancel,
		clock:   config.clock,
		logger:  config.logger,
		metrics: config.metrics,
		tracer:  config.tracer,
	}

	go func() {
		s.logger.InfoContext(ctx, "goroutine")

		if err := s.scanExecs(ctx, execs); err != nil {
			s.errChan <- err
		}
	}()

	go s.asyncExec(ctx, rebootExecs)

	go s.consumeErrors(ctx)

	return s, nil
}

// Next complies with the common selector interface, which would run a (set of) executors and return an error if raised.
//
// In an async approach, this method call will simply wait for the configured timeout duration and return the error(s)
// collected by its Selector.Error method call.
//
// Callers of Selector.Next should consider to use Selector.Error directly, instead.
func (s *Selector) Next(ctx context.Context) error {
	timer := time.NewTimer(s.timeout)

	select {
	case <-ctx.Done():
		s.cancel()

		return nil
	case <-timer.C:
		return s.Error()
	}
}

// Error will return any stored errors in the Selector, with a swap-with-nil approach, resetting the errors
// stored in the Selector.
//
// Errors are collected asynchronously and joined.
func (s *Selector) Error() error {
	stored := s.err.Swap(nil)
	if stored != nil && *stored != nil {
		return *stored
	}

	return nil
}

// Stop interrupts the (asnyc) Selector's runtime.
func (s *Selector) Stop() error {
	s.cancel()

	return s.Error()
}

func (s *Selector) scanExecs(ctx context.Context, e []*executor.Executable) error {
	if len(e) == 0 {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			s.asyncExec(ctx, e)
		}
	}
}

func (s *Selector) asyncExec(ctx context.Context, e []*executor.Executable) {
	ctx, span := s.tracer.Start(ctx, "Selector.asyncExec")
	defer span.End()

	s.metrics.IncSelectorSelectCalls(ctx)

	now := s.clock.Now()

	select {
	case <-ctx.Done():
		return
	default:
		s.logger.InfoContext(ctx, "selecting the next task")
		errExecs := make([]error, 0, len(e))

		for i := range e {
			// skip and remove nil executors
			if e[i] == nil {
				e = append(e[:i], e[i+1:]...)

				continue
			}

			if err := e[i].Exec(ctx, now); err != nil {
				err = fmt.Errorf("%s: %w", e[i].ID(), err)

				// join errors on a set of executor.Exec calls
				errExecs = append(errExecs, err)
			}

			// remove reboot executors after the first run
			if executor.IsReboot(e[i]) {
				e = append(e[:i], e[i+1:]...)
			}
		}

		// if there are errors, register them and send them to the error channel
		if len(errExecs) > 0 {
			err := errors.Join(errExecs...)
			s.errChan <- err

			s.metrics.IncSelectorSelectErrors(ctx)
			s.logger.ErrorContext(ctx, "executing next task asynchronously",
				slog.String("errors", err.Error()))
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}

		// if there are no other executors, return early
		if len(e) == 0 {
			return
		}
	}
}

func (s *Selector) consumeErrors(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-s.errChan:
			if !ok {
				return
			}

			s.storeError(err)
		}
	}
}

func (s *Selector) storeError(err error) {
	if err == nil {
		return
	}

	stored := s.err.Load()
	if stored == nil || *stored == nil {
		s.err.Store(&err)

		return
	}

	joined := errors.Join(*stored, err)
	s.err.Store(&joined)
}

func JoinSchedules(execs []Executor) ([]*executor.Executable, []*executor.Executable) {
	m := make([]*executor.Executable, 0, len(execs))
	r := make([]*executor.Executable, 0, len(execs))

	for i := range execs {
		if e, ok := execs[i].(*executor.Executable); ok && e != nil {
			if executor.IsReboot(e) {
				r = append(r, e)

				continue
			}

			m = append(m, e)
		}
	}

	return m, r
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
