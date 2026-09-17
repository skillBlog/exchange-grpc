package application_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/userservice/internal/infrastructure/ratelimit"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogin_success(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := sessionvalidation.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, nil)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "login@example.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute), nil)
	out, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "login@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	claims, err := accessTokens.Validate(out.AccessToken)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID == "" {
		t.Fatal("expected user id in claims")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "user" {
		t.Fatalf("Roles = %v, want [user]", claims.Roles)
	}
}

func TestLogin_invalidPassword(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := sessionvalidation.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, nil)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "login@example.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute), nil)
	_, err = login.Execute(context.Background(), application.LoginInput{
		Email:    "login@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
}

func TestLogin_rateLimited(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := sessionvalidation.NewRefreshTokenService(refreshRepo, 24*time.Hour)
	limiter := ratelimit.NewLoginLimiter(1, time.Minute)

	login := application.NewLogin(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, limiter, nil)

	_, err = login.Execute(context.Background(), application.LoginInput{
		Email:    "missing@example.com",
		Password: "Password1!",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("first error = %v, want ErrUnauthorized", err)
	}

	_, err = login.Execute(context.Background(), application.LoginInput{
		Email:    "missing@example.com",
		Password: "Password1!",
	})
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("second error = %v, want ErrRateLimited", err)
	}
}

func TestLogin_unknownUserIsLoggedWithoutPassword(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	login := newLoginForLogs(t, zap.New(core), memory.NewUserRepository())

	const password = "Password1!"
	_, err := login.Execute(context.Background(), application.LoginInput{
		Email:      "missing@example.com",
		Password:   password,
		ClientAddr: "10.0.0.8:443",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}

	assertFailedLoginLog(t, logs, "missing@example.com", "user-not-found", "10.0.0.8:443", password)
}

func TestLogin_badPasswordIsLoggedWithoutPassword(t *testing.T) {
	repo := memory.NewUserRepository()
	core, logs := observer.New(zapcore.WarnLevel)
	login := newLoginForLogs(t, zap.New(core), repo)

	register := application.NewRegister(
		repo,
		bcrypt.NewHasher(0),
		mustAccessTokens(t),
		sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour),
		nil,
	)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "login@example.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	const wrongPassword = "wrong-password"
	_, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "login@example.com",
		Password: wrongPassword,
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}

	assertFailedLoginLog(t, logs, "login@example.com", "bad-password", "", wrongPassword)
}

type countingHasher struct {
	inner application.PasswordHasher
	n     *int
}

func (h countingHasher) Hash(password string) (string, error) {
	return h.inner.Hash(password)
}

func (h countingHasher) Compare(hash, password string) error {
	*h.n++
	return h.inner.Compare(hash, password)
}

func TestLogin_unknownUserStillComparesPassword(t *testing.T) {
	var compares int
	login := application.NewLogin(
		memory.NewUserRepository(),
		countingHasher{inner: bcrypt.NewHasher(0), n: &compares},
		mustAccessTokens(t),
		sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour),
		ratelimit.NewLoginLimiter(10, time.Minute),
		nil,
	)
	_, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "missing@example.com",
		Password: "Password1!",
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
	if compares != 1 {
		t.Fatalf("Compare calls = %d, want 1", compares)
	}
}

func TestLogin_repoErrorStillComparesPassword(t *testing.T) {
	var compares int
	login := application.NewLogin(
		failingUserRepo{err: context.DeadlineExceeded},
		countingHasher{inner: bcrypt.NewHasher(0), n: &compares},
		mustAccessTokens(t),
		sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour),
		ratelimit.NewLoginLimiter(10, time.Minute),
		nil,
	)
	_, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "login@example.com",
		Password: "Password1!",
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want DeadlineExceeded", err)
	}
	if compares != 1 {
		t.Fatalf("Compare calls = %d, want 1", compares)
	}
}

type failingUserRepo struct {
	err error
}

func (r failingUserRepo) Save(context.Context, domain.User) error { return nil }

func (r failingUserRepo) GetByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, r.err
}

func (r failingUserRepo) GetByID(context.Context, string) (domain.User, error) {
	return domain.User{}, r.err
}

func newLoginForLogs(t *testing.T, log *zap.Logger, repo *memory.UserRepository) *application.Login {
	t.Helper()
	return application.NewLogin(
		repo,
		bcrypt.NewHasher(0),
		mustAccessTokens(t),
		sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour),
		ratelimit.NewLoginLimiter(10, time.Minute),
		log,
	)
}

func mustAccessTokens(t *testing.T) *sessionvalidation.TokenService {
	t.Helper()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	return accessTokens
}

func assertFailedLoginLog(t *testing.T, logs *observer.ObservedLogs, email, reason, clientAddr, password string) {
	t.Helper()
	entries := logs.FilterMessage("login failed").All()
	if len(entries) != 1 {
		t.Fatalf("login failed logs = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if _, ok := fields["email"]; ok {
		t.Fatal("raw email must not be logged")
	}
	wantHash := sha256Hex(email)
	if got, _ := fields["email_hash"].(string); got != wantHash {
		t.Fatalf("email_hash = %q, want %q", got, wantHash)
	}
	if got, _ := fields["reason"].(string); got != reason {
		t.Fatalf("reason = %q, want %q", got, reason)
	}
	if clientAddr != "" {
		if got, _ := fields["client_addr"].(string); got != clientAddr {
			t.Fatalf("client_addr = %q, want %q", got, clientAddr)
		}
	}
	if _, ok := fields["password"]; ok {
		t.Fatal("password must not be logged")
	}
	if strings.Contains(fmt.Sprint(fields), password) || strings.Contains(entries[0].Message, password) {
		t.Fatal("password must not appear in login failed log")
	}
	if strings.Contains(fmt.Sprint(fields), email) || strings.Contains(entries[0].Message, email) {
		t.Fatal("raw email must not appear in login failed log")
	}
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
