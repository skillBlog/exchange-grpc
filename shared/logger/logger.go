package logger

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const defaultAdminShutdownTimeout = 2 * time.Second

// New создаёт production zap-логгер с AtomicLevel (уровень из LOG_LEVEL).
func New() (*zap.Logger, zap.AtomicLevel, error) {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if raw := strings.TrimSpace(os.Getenv("LOG_LEVEL")); raw != "" {
		if err := SetLevel(level, raw); err != nil {
			return nil, zap.AtomicLevel{}, err
		}
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = level
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	log, err := cfg.Build()
	if err != nil {
		return nil, zap.AtomicLevel{}, err
	}
	return log, level, nil
}

// NewNop возвращает no-op логгер для тестов.
func NewNop() *zap.Logger {
	return zap.NewNop()
}

// SetLevel меняет уровень логирования в runtime.
func SetLevel(level zap.AtomicLevel, raw string) error {
	parsed, err := zapcore.ParseLevel(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("parse log level %q: %w", raw, err)
	}
	level.SetLevel(parsed)
	return nil
}

// ServeLevelAdmin поднимает HTTP endpoint AtomicLevel (GET/PUT), если addr не пустой.
// Пример: LOG_LEVEL_ADDR=:9090 → curl -X PUT localhost:9090 -d '{"level":"debug"}'
// Вызывающий код должен сделать ShutdownLevelAdmin до закрытия логгера.
func ServeLevelAdmin(addr string, level zap.AtomicLevel, log *zap.Logger) *http.Server {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.Handle("/", level)
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if log != nil {
				log.Warn("log level admin server stopped", zap.Error(err))
			}
		}
	}()
	if log != nil {
		log.Info("log level admin listening", zap.String("addr", addr))
	}
	return srv
}

// ShutdownLevelAdmin останавливает admin HTTP-сервер. nil безопасен.
func ShutdownLevelAdmin(srv *http.Server, log *zap.Logger) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAdminShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && log != nil {
		log.Warn("log level admin shutdown", zap.Error(err))
	}
}
