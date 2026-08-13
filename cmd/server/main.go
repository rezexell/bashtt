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

	//http
	createHandler := httptransport.NewCreateHandler(
		createService,
	)

	router := httptransport.NewRouter(
		logger,
		createHandler,
	)

	server := &http.Server{
		Addr:              cfg.HTTP.CreateAddr,
		Handler:           router.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	//graceful sd
	serverErr := make(chan error, 1)

	go func() {
		logger.Info(
			"HTTP server started",
			"address", cfg.HTTP.CreateAddr,
		)

		serverErr <- server.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErr:
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

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("server stopped")

	return nil
}
