package executor

import (
	"context"
	"errors"
	"sync"
	"time"
)

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

// Multi calls the Executor.Exec on a set of Executor that are meant to be executed at the same time (e.g. when
// scheduled for the same time).
//
// This call will execute the input (set of) action(s) in parallel, collecting any error(s) that the Executor(s) raises.
//
// The returned error is a joined error, for any failing executions. The executions are synchronized in a
// sync.WaitGroup, and are bound to the input context.Context's lifetime.
func Multi(ctx context.Context, now time.Time, execs ...Executor) error {
	errs := make([]error, 0, len(execs))

	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for i := range execs {
		e := execs[i]

		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := e.Exec(ctx, now); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return errors.Join(errs...)
}
