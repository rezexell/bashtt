package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rezexell/bashtt/internal/agent"
)

func main() {
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		),
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	watcher := agent.NewFanotifyWatcher(
		agent.FanotifyConfig{
			WatchDir: "/tmp/bashtt",
		},
	)

	sender := agent.NewCallbackSender(
		&http.Client{
			Timeout: 10 * time.Second,
		},
		"http://127.0.0.1:8081/callback",
	)

	a := agent.New(
		logger,
		watcher,
		sender,
	)

	if err := a.Run(ctx); err != nil {
		logger.Error(
			"agent stopped",
			"error", err,
		)
	}
}
