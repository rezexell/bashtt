package logging

import (
	"log/slog"
	"os"
)

type Config struct {
	Level slog.Level
	JSON  bool
}

func New(cfg Config) *slog.Logger {
	var handler slog.Handler

	options := &slog.HandlerOptions{
		Level: cfg.Level,
	}

	if cfg.JSON {
		handler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		handler = slog.NewTextHandler(os.Stdout, options)
	}

	return slog.New(handler)
}
