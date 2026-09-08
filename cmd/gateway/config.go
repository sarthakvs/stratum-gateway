package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr           string
	RedisAddr      string
	RedisPassword  string
	PostgresURL    string
	OllamaBaseURL  string
	RequestTimeout time.Duration
	DrainTimeout   time.Duration
	FailOpen       bool
}

func envStr(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}
func envDur(key string, def time.Duration) (time.Duration, error) {
	if val := os.Getenv(key); val == "" {
		return def, nil
	} else {
		d, err := time.ParseDuration(val)
		if err != nil {
			return def, fmt.Errorf("%s: %w", key, err)
		}
		return d, nil
	}
}
func envBool(key string, def bool) (bool, error) {
	if val := os.Getenv(key); val == "" {
		return def, nil
	} else {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return def, fmt.Errorf("%s: %w", key, err)
		}
		return b, nil
	}
}
func LoadConfig() (Config, error) {
	var errs []error

	reqTimeout, err := envDur("REQUEST_TIMEOUT", 60*time.Second)
	if err != nil {
		errs = append(errs, err)
	}
	drainTimeout, err := envDur("DRAIN_TIMEOUT", 30*time.Second)
	if err != nil {
		errs = append(errs, err)
	}
	failOpen, err := envBool("FAIL_OPEN", true)
	if err != nil {
		errs = append(errs, err)
	}

	cfg := Config{
		Addr:           envStr("ADDR", ":8080"),
		RedisAddr:      envStr("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  envStr("REDIS_PASSWORD", ""),
		PostgresURL:    envStr("POSTGRES_URL", "postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable"),
		OllamaBaseURL:  envStr("OLLAMA_BASE_URL", "http://localhost:11434"),
		RequestTimeout: reqTimeout,
		DrainTimeout:   drainTimeout,
		FailOpen:       failOpen,
	}
	if cfg.Addr == "" {
		errs = append(errs, errors.New("ADDR is empty"))
	}
	if cfg.RequestTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%s must be positive, got %s", "REQUEST_TIMEOUT", cfg.RequestTimeout))
	}
	if cfg.DrainTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%s must be positive, got %s", "DRAIN_TIMEOUT", cfg.DrainTimeout))
	}
	if ollamaCheck, err := url.Parse(cfg.OllamaBaseURL); err != nil {
		errs = append(errs, fmt.Errorf("OLLAMA_BASE_URL: invalid URL: %w", err))
	} else {
		if ollamaCheck.Host == "" {
			errs = append(errs, errors.New("OLLAMA_BASE_URL has no host"))
		}
		if ollamaCheck.Scheme == "" {
			errs = append(errs, fmt.Errorf("ollama's Scheme is empty"))
		}
	}
	return cfg, errors.Join(errs...)
}
