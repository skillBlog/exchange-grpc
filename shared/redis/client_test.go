package redis

import (
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func TestApplyConnLifetimeDefaults(t *testing.T) {
	parsed, err := goredis.ParseURL("redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("ParseURL() error = %v", err)
	}
	applyConnLifetimeDefaults(parsed)

	if parsed.ConnMaxIdleTime != defaultConnMaxIdleTime {
		t.Fatalf("ConnMaxIdleTime = %v, want %v", parsed.ConnMaxIdleTime, defaultConnMaxIdleTime)
	}
	if parsed.ConnMaxLifetime != defaultConnMaxLifetime {
		t.Fatalf("ConnMaxLifetime = %v, want %v", parsed.ConnMaxLifetime, defaultConnMaxLifetime)
	}
}

func TestWithConnLifetimeOptionsOverrideDefaults(t *testing.T) {
	parsed, err := goredis.ParseURL("redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("ParseURL() error = %v", err)
	}

	WithConnMaxIdleTime(2 * time.Minute)(parsed)
	WithConnMaxLifetime(time.Hour)(parsed)
	applyConnLifetimeDefaults(parsed)

	if parsed.ConnMaxIdleTime != 2*time.Minute {
		t.Fatalf("ConnMaxIdleTime = %v, want 2m", parsed.ConnMaxIdleTime)
	}
	if parsed.ConnMaxLifetime != time.Hour {
		t.Fatalf("ConnMaxLifetime = %v, want 1h", parsed.ConnMaxLifetime)
	}
}
