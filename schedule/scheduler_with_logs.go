package schedule

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/zalgonoise/micron/log"
)

type withLogs struct {
	s      Scheduler
	logger *slog.Logger
}

// Next calculates and returns the following scheduled time, from the input time.Time.
func (s withLogs) Next(ctx context.Context, now time.Time) time.Time {
	next := s.s.Next(ctx, now)

	s.logger.InfoContext(ctx, "next job", slog.Time("at", next))

	return next
}

// AddLogs decorates the input Scheduler with logging, using the input slog.Handler.
//
// If the input Scheduler is nil or a no-op Scheduler, a no-op Scheduler is returned. If the input slog.Handler is nil
// or a no-op handler, then a default slog.Handler is configured (a text handler printing to standard-error)
//
// If the input Scheduler is already a logged Scheduler, then this logged Scheduler is returned with the new handler as
// its logger's handler.
//
// Otherwise, the Scheduler is decorated with logging within a custom type that implements Scheduler.
func AddLogs(s Scheduler, handler slog.Handler) Scheduler {
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
