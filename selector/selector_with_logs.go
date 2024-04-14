package selector

import (
	"log/slog"

	"github.com/zalgonoise/micron/log"
)

// AddLogs replaces the input Selector's logger with a different one, using the input slog.Handler.
//
// If the input logger is nil, the Selector's logger will be set to be a no-op.
//
// If the input Selector is nil or a no-op Selector, a no-op Selector is returned.
//
// If the input Selector is a valid Selector, then its logger is replaced with a new one using the input handler.
//
// Otherwise, the Selector is returned as-is.
func AddLogs(s Selector, handler slog.Handler) Selector {
	if handler == nil || handler == log.NoOp() {
		handler = log.NoOp()
	}

	if s == nil || s == NoOp() {
		return NoOp()
	}

	switch sel := s.(type) {
	case *selector:
		sel.logger = slog.New(handler)

		return sel
	case *blockingSelector:
		sel.logger = slog.New(handler)

		return sel
	default:
		return s
	}
}
