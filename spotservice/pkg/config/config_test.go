package config_test

import (
	"testing"

	"github.com/exchange-grpc/spotservice/pkg/config"
)

func TestMemoryRateLimitFallbackAllowed(t *testing.T) {
	if !(config.Config{AppEnv: "dev"}).MemoryRateLimitFallbackAllowed() {
		t.Fatal("dev must allow in-memory rate limit fallback")
	}
	if (config.Config{AppEnv: "production"}).MemoryRateLimitFallbackAllowed() {
		t.Fatal("production must not allow in-memory rate limit fallback")
	}
}
