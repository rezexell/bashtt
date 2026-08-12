package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTP HTTPConfig
}

type HTTPConfig struct {
	CreateAddr   string
	CallbackAddr string
}

func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			CreateAddr:   getEnv("HTTP_CREATE_ADDR", ":8080"),
			CallbackAddr: getEnv("HTTP_CALLBACK_ADDR", ":8081"),
		},
	}

	if cfg.HTTP.CreateAddr == "" {
		return Config{}, fmt.Errorf("HTTP_CREATE_ADDR is empty")
	}

	if cfg.HTTP.CallbackAddr == "" {
		return Config{}, fmt.Errorf("HTTP_CALLBACK_ADDR is empty")
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
