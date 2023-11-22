package executor

import (
	"context"
	"errors"
	"sync"
)

// Multi calls the Executor.Exec on a set of Executor that are meant to be executed at the same time (e.g. when
// scheduled for the same time).
//
// This call will execute the input (set of) action(s) in parallel, collecting any error(s) that the Executor(s) raises.
//
// The returned error is a joined error, for any failing executions. The executions are synchronized in a
// sync.WaitGroup, and are bound to the input context.Context's lifetime.
func Multi(ctx context.Context, execs ...Executor) error {
	errs := make([]error, 0, len(execs))

	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}

	for i := range execs {
		e := execs[i]

		wg.Add(1)
		go func() {
			defer wg.Done()

			if err := e.Exec(ctx); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return errors.Join(errs...)
}
