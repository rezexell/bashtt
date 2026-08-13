package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP HTTPConfig
	SSH  SSHConfig
	Log  LogConfig
}

type HTTPConfig struct {
	CreateAddr   string
	CallbackAddr string
}

type SSHConfig struct {
	Port           int
	ConnectTimeout time.Duration
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

	cfg := Config{
		HTTP: HTTPConfig{
			CreateAddr: getEnv(
				"HTTP_CREATE_ADDR",
				":8080",
			),
		},

		SSH: SSHConfig{
			Port:           sshPort,
			ConnectTimeout: sshTimeout,
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
