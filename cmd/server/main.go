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
	"github.com/rezexell/bashtt/internal/agent"
	"github.com/rezexell/bashtt/internal/config"
	"github.com/rezexell/bashtt/internal/logging"
	"github.com/rezexell/bashtt/internal/repository/memory"
	"github.com/rezexell/bashtt/internal/service"
	"github.com/rezexell/bashtt/internal/ssh"
	templatesprovider "github.com/rezexell/bashtt/internal/templates"
	httptransport "github.com/rezexell/bashtt/internal/transport/http"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("load .env: %v", err)
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

	logger.Info("cfg", "HTTP_CREATE_ADDR:", cfg.HTTP.CreateAddr)

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

	machineRepository :=
		memory.NewMachineRepository()

	scriptRepository :=
		memory.NewScriptRepository()

	eventRepository :=
		memory.NewEventRepository()

	agentInstaller :=
		agent.NewNoopInstaller()

		//app
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
	)

	//http
	createHandler := httptransport.NewCreateHandler(
		createService,
	)

	callbackHandler := httptransport.NewCallbackHandler(
		callbackService,
	)

	healthHandler := httptransport.NewHealthHandler()

	router := httptransport.NewRouter(
		logger,
		createHandler,
		callbackHandler,
		healthHandler,
	)

	createServer := newHTTPServer(cfg.HTTP.CreateAddr, router.CreateHandler())

	callbackServer := newHTTPServer(cfg.HTTP.CallbackAddr, router.CallbackHandler())
	//graceful sd
	serverErrors := make(chan error, 2)

	go func() {
		logger.Info(
			"create API started",
			"address", cfg.HTTP.CreateAddr,
		)

		serverErrors <- createServer.ListenAndServe()
	}()

	go func() {
		logger.Info(
			"callback API started",
			"address", cfg.HTTP.CallbackAddr,
		)

		serverErrors <- callbackServer.ListenAndServe()
	}()

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
