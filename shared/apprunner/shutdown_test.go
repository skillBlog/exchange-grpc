package apprunner

import (
	"testing"
	"time"

	"github.com/exchange-grpc/shared/logger"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestGracefulStop_stopsServer(t *testing.T) {
	listener := bufconn.Listen(1024)
	server := googlegrpc.NewServer()
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	GracefulStop(server, time.Second, logger.NewNop())

	select {
	case err := <-errCh:
		if err != nil && err != googlegrpc.ErrServerStopped {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after GracefulStop")
	}
}

func TestGracefulStop_nilServer(t *testing.T) {
	GracefulStop(nil, time.Second, logger.NewNop())
}
