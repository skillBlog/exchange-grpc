package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/exchange-grpc/userservice/pkg/config"
)

func TestLoadConfig_invalidAccessTTL(t *testing.T) {
	t.Setenv("JWT_ACCESS_TTL", "abc")
	_, err := config.LoadConfig()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "JWT_ACCESS_TTL") {
		t.Fatalf("error = %v, want JWT_ACCESS_TTL", err)
	}
}

func TestMemoryRateLimitFallbackAllowed(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{env: "local", want: true},
		{env: "dev", want: true},
		{env: "development", want: true},
		{env: "production", want: false},
		{env: "prod", want: false},
		{env: "staging", want: false},
	}
	for _, tc := range tests {
		cfg := config.Config{AppEnv: tc.env}
		if got := cfg.MemoryRateLimitFallbackAllowed(); got != tc.want {
			t.Fatalf("APP_ENV=%s allowed = %v, want %v", tc.env, got, tc.want)
		}
	}
}

func TestLoadConfig_emptyAccessTTLUsesDefault(t *testing.T) {
	t.Setenv("JWT_ACCESS_TTL", "")
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Fatalf("AccessTokenTTL = %v, want 15m", cfg.AccessTokenTTL)
	}
}
