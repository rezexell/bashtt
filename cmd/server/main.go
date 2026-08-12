package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rezexell/bashtt/internal/config"
	"github.com/rezexell/bashtt/internal/logging"
	httptransport "github.com/rezexell/bashtt/internal/transport/http"
)

const (
	shutdownTimeout = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(logging.Config{
		Level: slog.LevelInfo,
		JSON:  false,
	})

	logger.Info("starting server")

	router := httptransport.NewRouter(logger)

	createServer := &http.Server{
		Addr:              cfg.HTTP.CreateAddr,
		Handler:           router.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	callbackServer := &http.Server{
		Addr:              cfg.HTTP.CallbackAddr,
		Handler:           router.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 2)

	go func() {
		logger.Info(
			"create API started",
			"address", createServer.Addr,
		)

		if err := createServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	go func() {
		logger.Info(
			"callback API started",
			"address", callbackServer.Addr,
		)

		if err := callbackServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		logger.Error(
			"HTTP server failed",
			"error", err,
		)

		return err

	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	logger.Info("shutting down servers")

	if err := createServer.Shutdown(shutdownCtx); err != nil {
		logger.Error(
			"failed to shutdown create API",
			"error", err,
		)
	}

	if err := callbackServer.Shutdown(shutdownCtx); err != nil {
		logger.Error(
			"failed to shutdown callback API",
			"error", err,
		)
	}

	logger.Info("server stopped")

	return nil
}
