package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP     HTTPConfig
	SSH      SSHConfig
	Postgres PostgresConfig
	Agent    AgentConfig
	Log      LogConfig
}

type HTTPConfig struct {
	CreateAddr   string
	CallbackAddr string
}

type SSHConfig struct {
	Port           int
	ConnectTimeout time.Duration
}

type PostgresConfig struct {
	URL string
}

type AgentConfig struct {
	LocalBinaryPath  string
	RemoteBinaryPath string
	WatchDir         string
	CallbackURL      string
	PIDFilePath      string
}

type LogConfig struct {
	Level string
	JSON  bool
}

func Load() (Config, error) {
	sshPort, err := getEnvInt("SSH_PORT", 22)
	if err != nil {
		return Config{}, fmt.Errorf("SSH_PORT: %w", err)
	}

	sshTimeout, err := getEnvDuration(
		"SSH_TIMEOUT",
		10*time.Second,
	)
	if err != nil {
		return Config{}, fmt.Errorf(
			"SSH_TIMEOUT: %w",
			err,
		)
	}

	jsonLogs, err := getEnvBool(
		"LOG_JSON",
		false,
	)
	if err != nil {
		return Config{}, fmt.Errorf(
			"LOG_JSON: %w",
			err,
		)
	}

	postgresURL := getEnv(
		"POSTGRES_URL",
		"postgres://bashtt:bashtt@localhost:5432/bashtt?sslmode=disable",
	)
	cfg := Config{
		HTTP: HTTPConfig{
			CreateAddr: getEnv(
				"HTTP_CREATE_ADDR",
				":8080",
			),
			CallbackAddr: getEnv(
				"HTTP_CALLBACK_ADDR",
				":8081",
			),
		},

		SSH: SSHConfig{
			Port:           sshPort,
			ConnectTimeout: sshTimeout,
		},

		Postgres: PostgresConfig{
			URL: postgresURL,
		},

		Agent: AgentConfig{
			LocalBinaryPath: getEnv(
				"AGENT_LOCAL_BINARY",
				"./bin/agent",
			),

			RemoteBinaryPath: getEnv(
				"AGENT_REMOTE_BINARY",
				"/tmp/bashtt/agent",
			),

			WatchDir: getEnv(
				"AGENT_WATCH_DIR",
				"/tmp/bashtt",
			),

			CallbackURL: getEnv(
				"AGENT_CALLBACK_URL",
				"http://host.docker.internal:8081/callback",
			),

			PIDFilePath: getEnv(
				"AGENT_PID_FILE",
				"/tmp/bashtt/agent.pid",
			),
		},

		Log: LogConfig{
			Level: getEnv(
				"LOG_LEVEL",
				"info",
			),
			JSON: jsonLogs,
		},
	}

	if cfg.HTTP.CreateAddr == "" {
		return Config{}, fmt.Errorf(
			"HTTP_CREATE_ADDR is empty",
		)
	}

	if cfg.HTTP.CallbackAddr == "" {
		return Config{}, fmt.Errorf(
			"HTTP_CALLBACK_ADDR is empty",
		)
	}

	if cfg.SSH.Port <= 0 || cfg.SSH.Port > 65535 {
		return Config{}, fmt.Errorf(
			"SSH_PORT must be between 1 and 65535",
		)
	}

	if cfg.SSH.ConnectTimeout <= 0 {
		return Config{}, fmt.Errorf(
			"SSH_TIMEOUT must be greater than zero",
		)
	}

	if cfg.Postgres.URL == "" {
		return Config{}, fmt.Errorf(
			"DATABASE_URL is empty",
		)
	}

	if cfg.Agent.LocalBinaryPath == "" {
		return Config{}, fmt.Errorf(
			"AGENT_BINARY_PATH is empty",
		)
	}

	if cfg.Agent.RemoteBinaryPath == "" {
		return Config{}, fmt.Errorf(
			"AGENT_REMOTE_BINARY_PATH is empty",
		)
	}

	if cfg.Agent.WatchDir == "" {
		return Config{}, fmt.Errorf(
			"AGENT_WATCH_DIR is empty",
		)
	}

	if cfg.Agent.CallbackURL == "" {
		return Config{}, fmt.Errorf(
			"AGENT_CALLBACK_URL is empty",
		)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)

	if value == "" {
		return fallback, nil
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid integer %q",
			value,
		)
	}

	return result, nil
}

func getEnvDuration(
	key string,
	fallback time.Duration,
) (time.Duration, error) {
	value := os.Getenv(key)

	if value == "" {
		return fallback, nil
	}

	result, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid duration %q",
			value,
		)
	}

	return result, nil
}

func getEnvBool(
	key string,
	fallback bool,
) (bool, error) {
	value := os.Getenv(key)

	if value == "" {
		return fallback, nil
	}

	result, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf(
			"invalid boolean %q",
			value,
		)
	}

	return result, nil
}
