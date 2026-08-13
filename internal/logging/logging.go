package logging

import (
	"log/slog"
	"os"
)

type Config struct {
	Level string
	JSON  bool
}

func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)

	var handler slog.Handler

	options := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.JSON {
		handler = slog.NewJSONHandler(
			os.Stdout,
			options,
		)
	} else {
		handler = slog.NewTextHandler(
			os.Stdout,
			options,
		)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug

	case "warn":
		return slog.LevelWarn

	case "error":
		return slog.LevelError

	case "info":
		fallthrough
	default:
		return slog.LevelInfo
	}
}
