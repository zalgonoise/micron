package schedule

import (
	"log/slog"

	"github.com/zalgonoise/micron/log"
)

// AddLogs replaces the input Scheduler's logger with a different one, using the input slog.Handler.
//
// If the input logger is nil, the Scheduler's logger will be set to be a no-op.
//
// If the input Scheduler is nil or a no-op Scheduler, a no-op Scheduler is returned.
//
// If the input Scheduler is a valid Scheduler, then its logger is replaced with a new one using the input handler.
//
// Otherwise, the Scheduler is returned as-is.
func AddLogs(s Scheduler, handler slog.Handler) Scheduler {
	if handler == nil || handler == log.NoOp() {
		handler = log.NoOp()
	}

	if s == nil || s == NoOp() {
		return NoOp()
	}

	sched, ok := s.(*CronSchedule)
	if !ok {
		return s
	}

	sched.logger = slog.New(handler)

	return sched
}
