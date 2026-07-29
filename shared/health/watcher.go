package health

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const defaultCheckInterval = 10 * time.Second

// Checker проверяет готовность зависимости сервиса.
type Checker func(ctx context.Context) error

// Watcher периодически обновляет gRPC health status.
type Watcher struct {
	server         *health.Server
	serviceName    string
	criticalChecks []Checker
	optionalChecks []Checker
	interval       time.Duration
	log            *zap.Logger
}

// NewWatcher создаёт health watcher.
// criticalChecks валят статус сервиса; optionalChecks только логируются (graceful degradation).
func NewWatcher(
	server *health.Server,
	serviceName string,
	interval time.Duration,
	log *zap.Logger,
	criticalChecks []Checker,
	optionalChecks ...Checker,
) *Watcher {
	if interval <= 0 {
		interval = defaultCheckInterval
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Watcher{
		server:         server,
		serviceName:    serviceName,
		criticalChecks: append([]Checker(nil), criticalChecks...),
		optionalChecks: append([]Checker(nil), optionalChecks...),
		interval:       interval,
		log:            log,
	}
}

// Run запускает периодические health checks до отмены контекста.
func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.update(ctx)
		}
	}
}

func (w *Watcher) update(ctx context.Context) {
	status := healthpb.HealthCheckResponse_SERVING
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	for _, check := range w.criticalChecks {
		if err := check(checkCtx); err != nil {
			status = healthpb.HealthCheckResponse_NOT_SERVING
			w.log.Error("critical health check failed", zap.Error(err))
			break
		}
	}

	for _, check := range w.optionalChecks {
		if err := check(checkCtx); err != nil {
			w.log.Warn("optional health check failed", zap.Error(err))
		}
	}

	// Только статус именованного сервиса — без глобального "" (ментор: не валить весь server status).
	if w.serviceName != "" {
		w.server.SetServingStatus(w.serviceName, status)
	}
}
