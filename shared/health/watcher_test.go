package health_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedhealth "github.com/exchange-grpc/shared/health"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const serviceName = "test.Service"

func TestWatcher_CheckFailsOnCritical(t *testing.T) {
	server := health.NewServer()
	boom := errors.New("db down")
	w := sharedhealth.NewWatcher(server, serviceName, time.Hour, time.Second, nil, []sharedhealth.Checker{
		func(context.Context) error { return boom },
	})

	if err := w.Check(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("Check() error = %v, want db down", err)
	}

	resp, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{Service: serviceName})
	if err != nil {
		t.Fatalf("status Check() error = %v", err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("status = %v, want NOT_SERVING", resp.GetStatus())
	}
}

func TestWatcher_CheckSetsServing(t *testing.T) {
	server := health.NewServer()
	w := sharedhealth.NewWatcher(server, serviceName, time.Hour, time.Second, nil, []sharedhealth.Checker{
		func(context.Context) error { return nil },
	})

	if err := w.Check(context.Background()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	resp, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{Service: serviceName})
	if err != nil {
		t.Fatalf("status Check() error = %v", err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("status = %v, want SERVING", resp.GetStatus())
	}
}

func TestWatcher_CheckOptionalFailureKeepsServing(t *testing.T) {
	server := health.NewServer()
	w := sharedhealth.NewWatcher(
		server,
		serviceName,
		time.Hour,
		time.Second,
		nil,
		[]sharedhealth.Checker{func(context.Context) error { return nil }},
		func(context.Context) error { return errors.New("redis down") },
	)

	if err := w.Check(context.Background()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	resp, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{Service: serviceName})
	if err != nil {
		t.Fatalf("status Check() error = %v", err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("status = %v, want SERVING", resp.GetStatus())
	}
}
