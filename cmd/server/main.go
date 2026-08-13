package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/rezexell/bashtt/internal/config"
	"github.com/rezexell/bashtt/internal/logging"
	postgresrepo "github.com/rezexell/bashtt/internal/repository/postgres"
	"github.com/rezexell/bashtt/internal/service"
	"github.com/rezexell/bashtt/internal/ssh"
	templatesprovider "github.com/rezexell/bashtt/internal/templates"
	httptransport "github.com/rezexell/bashtt/internal/transport/http"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("load .env: %v\n", err)
	}

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := logging.New(
		logging.Config{
			Level: cfg.Log.Level,
			JSON:  cfg.Log.JSON,
		},
	)

	logger.Info(
		"configuration loaded",
		"create_addr", cfg.HTTP.CreateAddr,
		"callback_addr", cfg.HTTP.CallbackAddr,
		"ssh_port", cfg.SSH.Port,
		"database", cfg.Postgres.URL,
		"agent_binary", cfg.Agent.LocalBinaryPath,
		"agent_remote_path", cfg.Agent.RemoteBinaryPath,
		"agent_watch_dir", cfg.Agent.WatchDir,
		"agent_callback_url", cfg.Agent.CallbackURL,
	)

	if err := run(logger, cfg); err != nil {
		logger.Error(
			"server stopped with error",
			"error", err,
		)

		os.Exit(1)
	}
}

func run(
	logger *slog.Logger,
	cfg config.Config,
) error {
	ctx := context.Background()

	// PostgreSQL.
	db, err := postgresrepo.New(
		ctx,
		cfg.Postgres.URL,
	)
	if err != nil {
		return fmt.Errorf(
			"connect to postgres: %w",
			err,
		)
	}

	defer db.Close()

	machineRepository := postgresrepo.NewMachineRepository(
		db,
	)

	scriptRepository := postgresrepo.NewScriptRepository(
		db,
	)

	eventRepository := postgresrepo.NewEventRepository(
		db,
	)

	templateProvider := templatesprovider.New()

	sshFactory := ssh.NewFactory(
		ssh.Config{
			Port:           cfg.SSH.Port,
			ConnectTimeout: cfg.SSH.ConnectTimeout,
		},
	)

	sshAdapter := service.NewSSHFactoryAdapter(
		sshFactory,
	)

	agentInstaller := service.NewAgentInstaller(
		service.AgentInstallerConfig{
			LocalBinaryPath:  cfg.Agent.LocalBinaryPath,
			RemoteBinaryPath: cfg.Agent.RemoteBinaryPath,
			WatchDir:         cfg.Agent.WatchDir,
			CallbackURL:      cfg.Agent.CallbackURL,
			PIDFilePath:      cfg.Agent.PIDFilePath,
		},
	)

	createService := service.NewCreateService(
		sshAdapter,
		templateProvider,
		machineRepository,
		scriptRepository,
		agentInstaller,
	)

	callbackService := service.NewCallbackService(
		logger,
		eventRepository,
		scriptRepository,
	)

	// HTTP handlers.
	createHandler := httptransport.NewCreateHandler(
		createService,
	)

	callbackHandler := httptransport.NewCallbackHandler(
		callbackService,
	)

	healthHandler := httptransport.NewHealthHandler()

	// HTTP router.
	router := httptransport.NewRouter(
		logger,
		createHandler,
		callbackHandler,
		healthHandler,
	)

	// HTTP servers.
	createServer := newHTTPServer(
		cfg.HTTP.CreateAddr,
		router.CreateHandler(),
	)

	callbackServer := newHTTPServer(
		cfg.HTTP.CallbackAddr,
		router.CallbackHandler(),
	)

	serverErrors := make(chan error, 2)

	// Create API.
	go func() {
		logger.Info(
			"create API started",
			"address", cfg.HTTP.CreateAddr,
		)

		serverErrors <- createServer.ListenAndServe()
	}()

	// Callback API.
	go func() {
		logger.Info(
			"callback API started",
			"address", cfg.HTTP.CallbackAddr,
		)

		serverErrors <- callbackServer.ListenAndServe()
	}()

	// Graceful shutdown.
	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err

	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := createServer.Shutdown(
		shutdownCtx,
	); err != nil {
		logger.Error(
			"create server shutdown failed",
			"error", err,
		)
	}

	if err := callbackServer.Shutdown(
		shutdownCtx,
	); err != nil {
		logger.Error(
			"callback server shutdown failed",
			"error", err,
		)
	}

	logger.Info("server stopped")

	return nil
}

func newHTTPServer(
	addr string,
	handler http.Handler,
) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
