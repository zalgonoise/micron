package selector

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type BlockingSelector struct {
	exec []Executor

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
func (s *BlockingSelector) Next(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "Selector.Select")
	defer span.End()

	s.metrics.IncSelectorSelectCalls(ctx)
	s.logger.InfoContext(ctx, "selecting the next task")

	// minStepDuration ensures that each execution is locked to the seconds mark and
	// a runner is not executed more than once per trigger.
	defer time.Sleep(minStepDuration)

	var err error

	switch len(s.exec) {
	case 0:
		err = ErrEmptyExecutorsList
	case 1:
		err = s.exec[0].Exec(ctx)
	default:
		err = s.next(ctx)[0].Exec(ctx)
	}

	if err != nil {
		s.metrics.IncSelectorSelectCalls(ctx)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		s.logger.ErrorContext(ctx, "no tasks configured for execution",
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}

func (s *BlockingSelector) next(ctx context.Context) []Executor {
	slices.SortFunc(s.exec, func(a, b Executor) int {
		return a.Next(ctx).Compare(b.Next(ctx))
	})

	if s.logger.Enabled(ctx, slog.LevelDebug) {
		times := make(map[string]string, len(s.exec))

		for i := range s.exec {
			times[s.exec[i].ID()] = s.exec[i].Next(ctx).Format(time.RFC3339)
		}

		s.logger.DebugContext(ctx, "selecting the next task",
			slog.Any("times", times),
		)
	}

	return s.exec
}
