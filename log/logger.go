package log

import (
	"log/slog"
	"os"

	"github.com/zalgonoise/cfg"
)

func New(h slog.Handler, options ...cfg.Option[Config]) *slog.Logger {
	config := cfg.New(options...)

	if h == nil {
		h = newHandler(config)
	}

	if config.withTraceID {
		h = NewSpanContextHandler(h, config.withSpanID)
	}

	return slog.New(h)
}

func newHandler(config Config) slog.Handler {
	switch config.format {
	case formatText:
		return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: config.source,
		})
	default:
		return slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: config.source,
		})
	}
}
