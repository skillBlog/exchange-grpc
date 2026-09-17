package grpc_test

import (
	"context"
	"net"
	"testing"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }

func TestClientAddrFromContext_prefersLeftmostForwardedFor(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-forwarded-for", " 203.0.113.10 , 10.0.0.1",
		"x-real-ip", "10.0.0.2",
	))
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: stubAddr("127.0.0.1:50051")})

	got := sharedgrpc.ClientAddrFromContext(ctx)
	if got != "203.0.113.10" {
		t.Fatalf("ClientAddrFromContext() = %q, want original client from X-Forwarded-For", got)
	}
}

func TestClientAddrFromContext_realIPFallback(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-real-ip", " 198.51.100.7 ",
	))
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: stubAddr("127.0.0.1:50051")})

	got := sharedgrpc.ClientAddrFromContext(ctx)
	if got != "198.51.100.7" {
		t.Fatalf("ClientAddrFromContext() = %q, want X-Real-IP", got)
	}
}

func TestClientAddrFromContext_peerFallback(t *testing.T) {
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: stubAddr("192.0.2.1:443")})
	got := sharedgrpc.ClientAddrFromContext(ctx)
	if got != "192.0.2.1:443" {
		t.Fatalf("ClientAddrFromContext() = %q, want peer.Addr", got)
	}
}

func TestClientAddrFromContext_empty(t *testing.T) {
	if got := sharedgrpc.ClientAddrFromContext(context.Background()); got != "" {
		t.Fatalf("ClientAddrFromContext() = %q, want empty", got)
	}
}

var _ net.Addr = stubAddr("")
