package grpc

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	metadataXForwardedFor = "x-forwarded-for"
	metadataXRealIP       = "x-real-ip"
)

// ClientAddrFromContext возвращает адрес клиента.
// Канон: левый (оригинальный) hop из X-Forwarded-For, иначе X-Real-IP, иначе peer.Addr.
func ClientAddrFromContext(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ip := firstForwardedAddr(md.Get(metadataXForwardedFor)); ip != "" {
			return ip
		}
		if ip := firstNonEmptyMetadata(md.Get(metadataXRealIP)); ip != "" {
			return ip
		}
	}
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return p.Addr.String()
	}
	return ""
}

func firstForwardedAddr(values []string) string {
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if addr := strings.TrimSpace(part); addr != "" {
				return addr
			}
		}
	}
	return ""
}

func firstNonEmptyMetadata(values []string) string {
	for _, value := range values {
		if addr := strings.TrimSpace(value); addr != "" {
			return addr
		}
	}
	return ""
}
