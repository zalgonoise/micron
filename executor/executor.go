package executor

import (
	"context"
	"errors"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/x/errs"

	"github.com/zalgonoise/micron/schedule"
)

const (
	cronAndLocAlloc = 2
	defaultID       = "micron.executor"
	bufferPeriod    = 100 * time.Millisecond

	errDomain = errs.Domain("micron/executor")

	ErrEmpty = errs.Kind("empty")

	ErrRunnerList = errs.Entity("runners list")
	ErrScheduler  = errs.Entity("scheduler")
)

var (
	ErrEmptyRunnerList = errs.WithDomain(errDomain, ErrEmpty, ErrRunnerList)
	ErrEmptyScheduler  = errs.WithDomain(errDomain, ErrEmpty, ErrScheduler)
)

// Runner describes a type that executes a job or task. It contains only one method, Run, that is called with a
// context as input and returns an error.
//
// Implementations of Runner only need to comply with this method, where the logic within Run is completely up to the
// actual implementation. These implementations need to be aware of the state of the input context.Context, which may
// denote cancellation or closure (e.g. with a timeout).
//
// The returned error denotes the success state of the execution. A nil error means that the execution was successful,
// where a non-nil error must signal a failed execution.
type Runner interface {
	// Run executes the job or task.
	//
	// This call takes in a context.Context which may be used to denote cancellation or closure (e.g. with a timeout)
	//
	// The returned error denotes the success state of the execution. A nil error means that the execution was successful,
	// where a non-nil error must signal a failed execution.
	Run(ctx context.Context) error
}

// Runnable is a custom type for any function that takes in a context.Context and returns an error. This type of
// function can be perceived as a Runner type. For that, this custom type will implement Runner by exposing a Run method
// that invokes the actual Runnable function.
type Runnable func(ctx context.Context) error

// Run executes the job or task.
//
// This call takes in a context.Context which may be used to denote cancellation or closure (e.g. with a timeout)
//
// The returned error denotes the success state of the execution. A nil error means that the execution was successful,
// where a non-nil error must signal a failed execution.
func (r Runnable) Run(ctx context.Context) error {
	if r == nil {
		return nil
	}

	return r(ctx)
}

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
	Exec(ctx context.Context) error
	// Next calls the Executor's underlying schedule.Scheduler Next method.
	Next(ctx context.Context) time.Time
	// ID returns this Executor's ID.
	ID() string
}

// Executable is an implementation of the Executor interface. It uses a schedule.Scheduler to mark the next job's
// execution time, and supports multiple Runner.
type Executable struct {
	id      string
	cron    schedule.Scheduler
	runners []Runner
}

// Next calls the Executor's underlying schedule.Scheduler Next method.
func (e Executable) Next(ctx context.Context) time.Time {
	return e.cron.Next(ctx, time.Now())
}

// Exec runs the task when on its scheduled time.
//
// For this, Exec leverages the Executor's underlying schedule.Scheduler to retrieve the job's next execution time,
// waits for it, and calls Runner.Run on each configured Runner. All raised errors are joined and returned at the end
// of this call.
func (e Executable) Exec(ctx context.Context) error {
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	now := time.Now()
	next := e.cron.Next(execCtx, now)
	timer := time.NewTimer(next.Sub(now))

	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:
			// avoid executing before it's time, as it may trigger repeated runs
			if preTriggerDuration := time.Since(next); preTriggerDuration > 0 {
				time.Sleep(preTriggerDuration + bufferPeriod)
			}

			runnerErrs := make([]error, 0, len(e.runners))

			for i := range e.runners {
				if err := e.runners[i].Run(ctx); err != nil {
					runnerErrs = append(runnerErrs, err)
				}
			}

			return errors.Join(runnerErrs...)
		}
	}
}

// ID returns this Executor's ID.
func (e Executable) ID() string {
	return e.id
}

// New creates an Executor with the input cfg.Option(s), also returning an error if raised.
//
// The minimum requirements to create an Executor is to supply at least one Runner, be it an implementation of
// this interface or as a Runnable using the WithRunners option, as well as a schedule.Scheduler using the
// WithScheduler option -- alternatively, callers can simply pass a cron string directly using the WithSchedule option.
//
// If an ID is not supplied, then the default ID of `micron.executor` is set.
func New(id string, options ...cfg.Option[*Config]) (Executor, error) {
	config := cfg.Set(new(Config), options...)

	exec, err := newExecutable(id, config)
	if err != nil {
		return noOpExecutor{}, err
	}

	if config.metrics != nil {
		exec = AddMetrics(exec, config.metrics)
	}

	if config.handler != nil {
		exec = AddLogs(exec, config.handler)
	}

	if config.tracer != nil {
		exec = AddTraces(exec, config.tracer)
	}

	return exec, nil
}

func newExecutable(id string, config *Config) (Executor, error) {
	// validate input
	if id == "" {
		id = defaultID
	}

	if len(config.runners) == 0 {
		return noOpExecutor{}, ErrEmptyRunnerList
	}

	if config.scheduler == nil && config.cron == "" {
		return noOpExecutor{}, ErrEmptyScheduler
	}

	var sched schedule.Scheduler

	switch {
	case config.scheduler != nil:
		// scheduler is provided, ignore cron string and location
		sched = config.scheduler
	default:
		// create a new scheduler from config
		opts := make([]cfg.Option[schedule.Config], 0, cronAndLocAlloc)

		if config.cron != "" {
			opts = append(opts, schedule.WithSchedule(config.cron))
		}

		if config.loc != nil {
			opts = append(opts, schedule.WithLocation(config.loc))
		}

		var err error

		sched, err = schedule.New(opts...)
		if err != nil {
			return noOpExecutor{}, err
		}
	}

	// return the object with the provided runners
	return Executable{
		id:      id,
		cron:    sched,
		runners: config.runners,
	}, nil
}

// NoOp returns a no-op Executor.
func NoOp() Executor {
	return noOpExecutor{}
}

type noOpExecutor struct{}

// Exec runs the task when on its scheduled time.
//
// This is a no-op call, it has no effect and the returned error is always nil.
func (e noOpExecutor) Exec(context.Context) error {
	return nil
}

// Next calls the Executor's underlying schedule.Scheduler Next method.
//
// This is a no-op call, it has no effect and the returned time is always zero.
func (e noOpExecutor) Next(_ context.Context) (t time.Time) {
	return t
}

// ID returns this Executor's ID.
//
// This is a no-op call, it has no effect and the returned string is always empty.
func (e noOpExecutor) ID() string {
	return ""
}
