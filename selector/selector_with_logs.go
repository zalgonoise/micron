package selector

import (
	"context"
	"log/slog"
	"os"

	"github.com/zalgonoise/micron/log"
)

type withLogs struct {
	s      Selector
	logger *slog.Logger
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
func (s withLogs) Next(ctx context.Context) error {
	s.logger.InfoContext(ctx, "selecting the next task")

	if err := s.s.Next(ctx); err != nil {
		s.logger.ErrorContext(ctx, "failed to select and execute the next task", slog.String("error", err.Error()))

		return err
	}

	return nil
}

// AddLogs decorates the input Selector with logging, using the input slog.Handler.
//
// If the input Selector is nil or a no-op Selector, a no-op Selector is returned. If the input slog.Handler is nil or a
// no-op handler, then a default slog.Handler is configured (a text handler printing to standard-error)
//
// If the input Selector is already a logged Selector, then this logged Selector is returned with the new handler as its
// logger's handler.
//
// Otherwise, the Selector is decorated with logging within a custom type that implements Selector.
func AddLogs(s Selector, handler slog.Handler) Selector {
	if s == nil || s == NoOp() {
		return NoOp()
	}

	if handler == nil || handler == log.NoOp() {
		handler = slog.NewTextHandler(os.Stderr, nil)
	}

	if logs, ok := s.(withLogs); ok {
		logs.logger = slog.New(handler)

		return logs
	}

	return withLogs{
		s:      s,
		logger: slog.New(handler),
	}
}
