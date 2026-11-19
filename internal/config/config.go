// Package config reads all settings from environment variables in one place,
// so the rest of the program never calls os.Getenv.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port        string // TCP port to listen on, default "8080"
	DatabaseURL string // required, e.g. postgres://user:pass@localhost:5432/db?sslmode=disable
}

// Load reads the configuration from the real environment.
func Load() (Config, error) { return load(os.Getenv) }

// load takes the lookup function as a parameter so tests can pass a fake environment.
func load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:        getenv("PORT"),
		DatabaseURL: getenv("DATABASE_URL"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if n, err := strconv.Atoi(cfg.Port); err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("PORT must be a number between 1 and 65535, got %q", cfg.Port)
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (copy .env.example to .env and fill it in)")
	}
	return cfg, nil
}
