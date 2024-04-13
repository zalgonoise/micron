package micron

import (
	"log/slog"

	"github.com/zalgonoise/micron/log"
)

// AddLogs decorates the input Runtime with logging, using the input slog.Handler.
//
// If the input logger is nil, the Runtime's logger will be set to be a no-op.
//
// If the input Runtime is nil or a no-op Runtime, a no-op Runtime is returned. If the input slog.Handler is nil or a
// no-op handler, then a default slog.Handler is configured (a text handler printing to standard-error)
//
// If the input Runtime is already a logged Runtime, then this logged Runtime is returned with the new handler as its
// logger's handler.
//
// Otherwise, the Runtime is decorated with logging within a custom type that implements Runtime.
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
