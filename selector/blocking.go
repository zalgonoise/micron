package selector

import (
	"context"
	"time"

	"github.com/zalgonoise/micron/executor"
)

type blockingSelector struct {
	exec []executor.Executor
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
func (s blockingSelector) Next(ctx context.Context) error {
	// minStepDuration ensures that each execution is locked to the seconds mark and
	// a runner is not executed more than once per trigger.
	defer time.Sleep(minStepDuration)

	switch len(s.exec) {
	case 0:
		return ErrEmptyExecutorsList
	case 1:
		return s.exec[0].Exec(ctx)
	default:
		return executor.Multi(ctx, s.next(ctx)...)
	}
}

func (s blockingSelector) next(ctx context.Context) []executor.Executor {
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
