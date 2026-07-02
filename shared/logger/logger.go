package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New создаёт production-ready zap-логгер с выводом в stdout.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}

// NewNop возвращает no-op логгер для тестов.
func NewNop() *zap.Logger {
	return zap.NewNop()
}
