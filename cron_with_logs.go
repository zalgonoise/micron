package micron

import (
	"context"
	"log/slog"
	"os"

	"github.com/zalgonoise/micron/log"
)

type withLogs struct {
	r      Runtime
	logger *slog.Logger
}

// Run kicks-off the cron module using the input context.Context.
//
// This is a blocking call that should be executed in a goroutine. The input context.Context can be leveraged to
// define when should the cron Runtime be halted, for example with context cancellation or timeout.
//
// Any error raised within a Run cycle is channeled to the Runtime errors channel, accessible with the Err method.
func (c withLogs) Run(ctx context.Context) {
	c.logger.InfoContext(ctx, "starting cron")
	c.r.Run(ctx)
	c.logger.InfoContext(ctx, "closing cron")
}

// Err returns a receive-only errors channel, allowing the caller to consumer any errors raised during the execution
// of cron jobs.
//
// It is the responsibility of the caller to consume these errors appropriately, within the logic of their app.
func (c withLogs) Err() <-chan error {
	return c.r.Err()
}

// AddLogs decorates the input Runtime with logging, using the input slog.Handler.
//
// If the input Runtime is nil or a no-op Runtime, a no-op Runtime is returned. If the input slog.Handler is nil or a
// no-op handler, then a default slog.Handler is configured (a text handler printing to standard-error)
//
// If the input Runtime is already a logged Runtime, then this logged Runtime is returned with the new handler as its
// logger's handler.
//
// Otherwise, the Runtime is decorated with logging within a custom type that implements Runtime.
func AddLogs(r Runtime, handler slog.Handler) Runtime {
	if r == nil || r == NoOp() {
		return NoOp()
	}

	if handler == nil || handler == log.NoOp() {
		handler = slog.NewTextHandler(os.Stderr, nil)
	}

	if logs, ok := r.(withLogs); ok {
		logs.logger = slog.New(handler)

		return logs
	}

	return withLogs{
		r:      r,
		logger: slog.New(handler),
	}
}
