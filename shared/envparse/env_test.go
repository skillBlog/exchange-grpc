package envparse_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/envparse"
)

func TestLoader_DurationUnsetUsesDefault(t *testing.T) {
	t.Setenv("ENVPARSE_TEST_DURATION_UNSET", "1s")
	if err := os.Unsetenv("ENVPARSE_TEST_DURATION_UNSET"); err != nil {
		t.Fatalf("Unsetenv() error = %v", err)
	}
	var env envparse.Loader
	got := env.Duration("ENVPARSE_TEST_DURATION_UNSET", 15*time.Minute)
	if err := env.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
	if got != 15*time.Minute {
		t.Fatalf("Duration() = %v, want 15m", got)
	}
}

func TestLoader_DurationEmptyUsesDefault(t *testing.T) {
	t.Setenv("ENVPARSE_TEST_DURATION_EMPTY", "  ")
	var env envparse.Loader
	got := env.Duration("ENVPARSE_TEST_DURATION_EMPTY", 15*time.Minute)
	if err := env.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
	if got != 15*time.Minute {
		t.Fatalf("Duration() = %v, want 15m", got)
	}
}

func TestLoader_DurationValid(t *testing.T) {
	t.Setenv("ENVPARSE_TEST_DURATION_OK", "30s")
	var env envparse.Loader
	got := env.Duration("ENVPARSE_TEST_DURATION_OK", 15*time.Minute)
	if err := env.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
	if got != 30*time.Second {
		t.Fatalf("Duration() = %v, want 30s", got)
	}
}

func TestLoader_DurationInvalidReturnsNamedError(t *testing.T) {
	t.Setenv("ENVPARSE_TEST_DURATION_BAD", "abc")
	var env envparse.Loader
	_ = env.Duration("ENVPARSE_TEST_DURATION_BAD", 15*time.Minute)
	err := env.Err()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "ENVPARSE_TEST_DURATION_BAD") {
		t.Fatalf("error = %v, want variable name", err)
	}
	if !strings.Contains(err.Error(), "abc") {
		t.Fatalf("error = %v, want raw value", err)
	}
}

func TestLoader_IntInvalidReturnsNamedError(t *testing.T) {
	t.Setenv("ENVPARSE_TEST_INT_BAD", "nope")
	var env envparse.Loader
	_ = env.Int("ENVPARSE_TEST_INT_BAD", 10)
	err := env.Err()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "ENVPARSE_TEST_INT_BAD") {
		t.Fatalf("error = %v, want variable name", err)
	}
}

func TestLoader_IntValid(t *testing.T) {
	t.Setenv("ENVPARSE_TEST_INT_OK", "7")
	var env envparse.Loader
	got := env.Int("ENVPARSE_TEST_INT_OK", 3)
	if err := env.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
	if got != 7 {
		t.Fatalf("Int() = %d, want 7", got)
	}
}

func TestLoader_keepsFirstError(t *testing.T) {
	t.Setenv("ENVPARSE_TEST_FIRST", "bad")
	t.Setenv("ENVPARSE_TEST_SECOND", "also-bad")
	var env envparse.Loader
	_ = env.Duration("ENVPARSE_TEST_FIRST", time.Second)
	_ = env.Int("ENVPARSE_TEST_SECOND", 1)
	err := env.Err()
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "ENVPARSE_TEST_FIRST") {
		t.Fatalf("error = %v, want first variable", err)
	}
	if strings.Contains(err.Error(), "ENVPARSE_TEST_SECOND") {
		t.Fatalf("error = %v, must keep first error", err)
	}
}
