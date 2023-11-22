package log

import (
	"github.com/zalgonoise/cfg"
)

const (
	formatJSON = iota
	formatText
)

type Config struct {
	format int
	source bool

	withTraceID bool
	withSpanID  bool
}

func AsText() cfg.Option[Config] {
	return cfg.Register(func(config Config) Config {
		config.format = formatText

		return config
	})
}

func AsJSON() cfg.Option[Config] {
	return cfg.Register(func(config Config) Config {
		config.format = formatJSON

		return config
	})
}

func WithSource() cfg.Option[Config] {
	return cfg.Register(func(config Config) Config {
		config.source = true

		return config
	})
}

func WithTraceContext(withSpanID bool) cfg.Option[Config] {
	return cfg.Register(func(config Config) Config {
		config.withTraceID = true
		config.withSpanID = withSpanID

		return config
	})
}
