package grpc

import (
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestFirstAuthorizationValue_usesMetadataGet(t *testing.T) {
	md := metadata.Pairs("Authorization", "  Bearer token  ", "x-request-id", "req")
	got, ok := firstAuthorizationValue(md)
	if !ok {
		t.Fatal("expected authorization value")
	}
	if got != "Bearer token" {
		t.Fatalf("got %q, want trimmed Bearer token", got)
	}
}

func TestFirstAuthorizationValue_skipsBlank(t *testing.T) {
	md := metadata.MD{MetadataAuthorization: []string{"  ", "", "Bearer ok"}}
	got, ok := firstAuthorizationValue(md)
	if !ok {
		t.Fatal("expected authorization value")
	}
	if got != "Bearer ok" {
		t.Fatalf("got %q, want Bearer ok", got)
	}
}
