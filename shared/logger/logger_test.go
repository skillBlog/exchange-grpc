package logger_test

import (
	"testing"

	"github.com/exchange-grpc/shared/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestSetLevel(t *testing.T) {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if err := logger.SetLevel(level, "debug"); err != nil {
		t.Fatalf("SetLevel() error = %v", err)
	}
	if got := level.Level(); got != zapcore.DebugLevel {
		t.Fatalf("level = %v, want debug", got)
	}
}

func TestSetLevel_invalid(t *testing.T) {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if err := logger.SetLevel(level, "nope"); err == nil {
		t.Fatal("expected error for invalid level")
	}
}
