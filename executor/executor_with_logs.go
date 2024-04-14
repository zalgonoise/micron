package executor

import (
	"log/slog"

	"github.com/zalgonoise/micron/log"
)

// AddLogs replaces the input Executor's logger with a different one, using the input slog.Handler.
//
// If the input logger is nil, the Executor's logger will be set to be a no-op.
//
// If the input Executor is nil or a no-op Executor, a no-op Executor is returned.
//
// If the input Executor is an Executable type, then its logger is replaced with the new one using the input handler.
//
// Otherwise, the Executor is returned as-is.
func AddLogs(e Executor, handler slog.Handler) Executor {
	if handler == nil || handler == log.NoOp() {
		handler = log.NoOp()
	}

	if e == nil || e == NoOp() {
		return NoOp()
	}

	executable, ok := e.(*Executable)
	if !ok {
		return e
	}

	executable.logger = slog.New(handler)

	return executable
}
