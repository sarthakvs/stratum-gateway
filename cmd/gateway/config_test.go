package main

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	for _, k := range []string{
		"ADDR", "REDIS_ADDR",
		"REDIS_PASSWORD", "POSTGRES_URL",
		"OLLAMA_BASE_URL", "REQUEST_TIMEOUT",
		"DRAIN_TIMEOUT", "FAIL_OPEN",
	} {
		t.Setenv(k, "")
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.RequestTimeout != 60*time.Second {
		t.Errorf("RequestTimeout = %v, want %v", cfg.RequestTimeout, 60*time.Second)
	}
	if cfg.DrainTimeout != 30*time.Second {
		t.Errorf("DrainTimeout = %v, want %v", cfg.DrainTimeout, 30*time.Second)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("RedisAddr = %q,want %q", cfg.RedisAddr, "localhost:6379")
	}
	if cfg.PostgresURL != "postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable" {
		t.Errorf("PostgresURL = %q,want %q", cfg.PostgresURL, "postgres://gateway:gateway@localhost:5432/gateway?sslmode=disable")
	}
	if cfg.OllamaBaseURL != "http://localhost:11434" {
		t.Errorf("OllamaBaseURL = %q,want %q", cfg.OllamaBaseURL, "http://localhost:11434")
	}
	if !cfg.FailOpen {
		t.Errorf("FailOpen = %v,want %v", cfg.FailOpen, true)
	}
	if cfg.RedisPassword != "" {
		t.Errorf("RedisPassword = %q,want %q", cfg.RedisPassword, "")
	}
}

func TestLoadConfig_Invalid(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		value           string
		wantErrContains string
	}{
		{"unparsable duration", "REQUEST_TIMEOUT", "not-a-duration", "REQUEST_TIMEOUT"},
		{"negative duration", "REQUEST_TIMEOUT", "-5s", "REQUEST_TIMEOUT"},
		{"bad bool", "FAIL_OPEN", "maybe", "FAIL_OPEN"},
		{"url with no host", "OLLAMA_BASE_URL", "localhost:11434", "OLLAMA_BASE_URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)
			_, err := LoadConfig()
			if err == nil {
				t.Fatalf("LoadConfig() = nil error, want an error")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("error = %q, want it to mention %q", err, tt.wantErrContains)
			}
		})
	}
}


