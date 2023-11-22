package tracing

import (
	"time"

	"github.com/zalgonoise/cfg"
)

type Config struct {
	timeout time.Duration

	username string
	password string
}

func WithTimeout(dur time.Duration) cfg.Option[Config] {
	if dur < 0 {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.timeout = dur

		return config
	})
}

func WithBasicAuth(username, password string) cfg.Option[Config] {
	if username == "" && password == "" {
		return cfg.NoOp[Config]{}
	}

	return cfg.Register(func(config Config) Config {
		config.username = username
		config.password = password

		return config
	})
}
