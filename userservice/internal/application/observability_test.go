package application_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/userservice/internal/infrastructure/ratelimit"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogin_successAuditOmitsSecrets(t *testing.T) {
	repo := memory.NewUserRepository()
	core, logs := observer.New(zapcore.InfoLevel)
	register := application.NewRegister(repo, bcrypt.NewHasher(0), mustAccessTokens(t), sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour), nil)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "audit@example.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(
		repo,
		bcrypt.NewHasher(0),
		mustAccessTokens(t),
		sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour),
		ratelimit.NewLoginLimiter(10, time.Minute),
		zap.New(core),
	)
	out, err := login.Execute(context.Background(), application.LoginInput{
		Email:      "audit@example.com",
		Password:   "Password1!",
		ClientAddr: "203.0.113.10",
	})
	if err != nil {
		t.Fatalf("login error = %v", err)
	}

	entries := logs.FilterMessage("user logged in").All()
	if len(entries) != 1 {
		t.Fatalf("audit logs = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if got, _ := fields["user_id"].(string); got == "" {
		t.Fatal("expected user_id in audit log")
	}
	if got, _ := fields["client_addr"].(string); got != "203.0.113.10" {
		t.Fatalf("client_addr = %q", got)
	}
	dump := fmt.Sprint(fields, entries[0].Message)
	if strings.Contains(dump, "audit@example.com") {
		t.Fatal("raw email must not appear in audit log")
	}
	if strings.Contains(fmt.Sprint(fields), out.AccessToken) || strings.Contains(fmt.Sprint(fields), out.RefreshToken) {
		t.Fatal("tokens must not appear in audit log")
	}
}

func TestRegister_successAuditOmitsEmail(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	register := application.NewRegister(
		memory.NewUserRepository(),
		bcrypt.NewHasher(0),
		mustAccessTokens(t),
		sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour),
		zap.New(core),
	)
	out, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "new@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("register error = %v", err)
	}
	entries := logs.FilterMessage("user registered").All()
	if len(entries) != 1 {
		t.Fatalf("audit logs = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if got, _ := fields["user_id"].(string); got != out.UserID {
		t.Fatalf("user_id = %q, want %q", got, out.UserID)
	}
	if strings.Contains(fmt.Sprint(fields), "new@example.com") {
		t.Fatal("raw email must not appear in register audit log")
	}
}

func TestLogout_createsSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	logout := application.NewLogout(sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), time.Hour), nil)
	if err := logout.Execute(context.Background(), application.LogoutInput{RefreshToken: strings.Repeat("a", 64)}); err != nil {
		t.Fatalf("logout error = %v", err)
	}
	if !hasSpan(exporter, "user.Logout") {
		t.Fatal("expected user.Logout span")
	}
}

func TestLogin_rateLimitAndPasswordSpans(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
	})

	login := application.NewLogin(
		memory.NewUserRepository(),
		bcrypt.NewHasher(0),
		mustAccessTokens(t),
		sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour),
		ratelimit.NewLoginLimiter(10, time.Minute),
		nil,
	)
	_, _ = login.Execute(context.Background(), application.LoginInput{
		Email:    "missing@example.com",
		Password: "Password1!",
	})

	for _, name := range []string{"user.Login", "user.loginRateLimit", "user.comparePassword"} {
		if !hasSpan(exporter, name) {
			t.Fatalf("missing span %s", name)
		}
	}
}

func hasSpan(exporter *tracetest.InMemoryExporter, name string) bool {
	for _, span := range exporter.GetSpans() {
		if span.Name == name {
			return true
		}
	}
	return false
}
