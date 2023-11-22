package executor

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/zalgonoise/micron/log"
)

type withLogs struct {
	e      Executor
	logger *slog.Logger
}

// Exec runs the task when on its scheduled time.
//
// For this, Exec leverages the Executor's underlying schedule.Scheduler to retrieve the job's next execution time,
// waits for it, and calls Runner.Run on each configured Runner. All raised errors are joined and returned at the end
// of this call.
func (e withLogs) Exec(ctx context.Context) error {
	id := slog.String("id", e.e.ID())

	e.logger.InfoContext(ctx, "executing task", id)

	err := e.e.Exec(ctx)
	if err != nil {
		e.logger.WarnContext(ctx, "task raised an error", id, slog.String("error", err.Error()))
	}

	return err
}

// Next calls the Executor's underlying schedule.Scheduler Next method.
func (e withLogs) Next(ctx context.Context) time.Time {
	next := e.e.Next(ctx)

	e.logger.InfoContext(ctx, "next job",
		slog.String("id", e.e.ID()),
		slog.Time("at", next),
	)

	return next
}

// ID returns this Executor's ID.
func (e withLogs) ID() string {
	return e.e.ID()
}

// AddLogs decorates the input Executor with logging, using the input slog.Handler.
//
// If the input Executor is nil or a no-op Executor, a no-op Executor is returned. If the input slog.Handler is nil or a
// no-op handler, then a default slog.Handler is configured (a text handler printing to standard-error)
//
// If the input Executor is already a logged Executor, then this logged Executor is returned with the new handler as its
// logger's handler.
//
// Otherwise, the Executor is decorated with logging within a custom type that implements Executor.
func AddLogs(e Executor, handler slog.Handler) Executor {
	if e == nil || e == NoOp() {
		return NoOp()
	}

	if handler == nil || handler == log.NoOp() {
		handler = slog.NewTextHandler(os.Stderr, nil)
	}

	if logs, ok := e.(withLogs); ok {
		logs.logger = slog.New(handler)

		return logs
	}

	return withLogs{
		e:      e,
		logger: slog.New(handler),
	}
}
