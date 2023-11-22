package log

import (
	"context"
	"log/slog"
)

func NoOp() slog.Handler {
	return noOpHandler{}
}

type noOpHandler struct{}

func (noOpHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (noOpHandler) Handle(context.Context, slog.Record) error { return nil }
func (h noOpHandler) WithAttrs([]slog.Attr) slog.Handler      { return h }
func (h noOpHandler) WithGroup(string) slog.Handler           { return h }
