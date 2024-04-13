package micron

import (
	"log/slog"

	"github.com/zalgonoise/micron/log"
)

// AddLogs replaces the input Runtime's logger with a different one, using the input slog.Handler.
//
// If the input logger is nil, the Runtime's logger will be set to be a no-op.
//
// If the input Runtime is nil or a no-op Runtime, a no-op Runtime is returned.
//
// If the input Runtime is a valid Runtime, then its logger is replaced with a new one using the input handler.
//
// Otherwise, the Runtime is returned as-is.
func AddLogs(r Runtime, handler slog.Handler) Runtime {
	if handler == nil || handler == log.NoOp() {
		handler = log.NoOp()
	}

	if r == nil || r == NoOp() {
		return NoOp()
	}

	cronRuntime, ok := r.(runtime)
	if !ok {
		return r
	}

	cronRuntime.logger = slog.New(handler)

	return cronRuntime
}
