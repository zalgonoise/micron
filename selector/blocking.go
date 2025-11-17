package selector

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/zalgonoise/cfg"
	"github.com/zalgonoise/micron/v3/executor"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type BlockingSelector struct {
	exec        []*executor.Executable
	rebootExecs []*executor.Executable
	clock       Clock

	logger  *slog.Logger
	metrics Metrics
	tracer  trace.Tracer
}

func NewBlockingSelector(options ...cfg.Option[*Config]) (*BlockingSelector, error) {
	config := cfg.Set(defaultConfig(), options...)

	if len(config.exec) == 0 {
		return nil, ErrEmptyExecutorsList
	}

	execs, rebootExecs := JoinSchedules(config.exec)

	s := &BlockingSelector{
		exec:        execs,
		rebootExecs: rebootExecs,
		clock:       config.clock,
		logger:      config.logger,
		metrics:     config.metrics,
		tracer:      config.tracer,
	}

	if len(rebootExecs) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), config.timeout)
		defer cancel()

		s.runRebootExecs(ctx)
	}

	return s, nil
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

	var (
		err error
		now = s.clock.Now()
	)

	switch len(s.exec) {
	case 0:
		err = ErrEmptyExecutorsList
	case 1:
		s.metrics.IncSelectorSelectCalls(ctx)

		err = s.exec[0].Exec(ctx, now)
	default:
		s.metrics.IncSelectorSelectCalls(ctx)

		execs := s.next(ctx, now)

		errs := make([]error, 0, len(execs))
		for _, e := range execs {
			if err := e.Exec(ctx, now); err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			err = errors.Join(errs...)
		}
	}

	if err != nil {
		s.metrics.IncSelectorSelectErrors(ctx)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		s.logger.ErrorContext(ctx, "no tasks configured for execution",
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}

func (s *BlockingSelector) Error() error { return nil }

func (s *BlockingSelector) Stop() error { return nil }

func (s *BlockingSelector) next(ctx context.Context, now time.Time) []*executor.Executable {
	if len(s.exec) == 0 {
		return nil
	}

	var (
		shortest time.Duration
		execs    = make([]*executor.Executable, 0, len(s.exec))
	)

	for _, e := range s.exec {
		next := e.Next(ctx, now)

		switch {
		case next < 0:
			continue
		case next == shortest:
			execs = append(execs, e)
		case shortest == 0, next < shortest:
			shortest = next
			execs = make([]*executor.Executable, 0, len(s.exec))
			execs = append(execs, e)
		}
	}

	return execs
}

func (s *BlockingSelector) runRebootExecs(ctx context.Context) {
	ctx, span := s.tracer.Start(ctx, "BlockingSelector.runRebootExecs")
	defer span.End()

	s.metrics.IncSelectorSelectCalls(ctx)

	var (
		errs = make([]error, 0, len(s.rebootExecs))
		now  = s.clock.Now()
	)

	s.logger.InfoContext(ctx, "running reboot executors")

	select {
	case <-ctx.Done():
		return
	default:
		for _, e := range s.rebootExecs {
			if err := e.Exec(ctx, now); err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			err := errors.Join(errs...)

			s.metrics.IncSelectorSelectErrors(ctx)
			s.logger.ErrorContext(ctx, "executing reboot tasks",
				slog.String("errors", err.Error()))
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
	}
}
