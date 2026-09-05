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

	applyOptions(parsed, WithConnMaxIdleTime(2*time.Minute), WithConnMaxLifetime(time.Hour))

	if parsed.ConnMaxIdleTime != 2*time.Minute {
		t.Fatalf("ConnMaxIdleTime = %v, want 2m", parsed.ConnMaxIdleTime)
	}
	if parsed.ConnMaxLifetime != time.Hour {
		t.Fatalf("ConnMaxLifetime = %v, want 1h", parsed.ConnMaxLifetime)
	}
}

func TestWithConnLifetimeZeroMeansNoLimit(t *testing.T) {
	parsed, err := goredis.ParseURL("redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("ParseURL() error = %v", err)
	}

	applyOptions(parsed, WithConnMaxIdleTime(0), WithConnMaxLifetime(0))

	if parsed.ConnMaxIdleTime != 0 {
		t.Fatalf("ConnMaxIdleTime = %v, want 0 (no limit)", parsed.ConnMaxIdleTime)
	}
	if parsed.ConnMaxLifetime != 0 {
		t.Fatalf("ConnMaxLifetime = %v, want 0 (no limit)", parsed.ConnMaxLifetime)
	}
}
