package application_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/roles"
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

func TestRefreshToken_success(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := sessionvalidation.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, nil)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "refresh@example.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute), nil)
	loginOut, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "refresh@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("login error = %v", err)
	}

	refreshUC := application.NewRefreshToken(repo, accessTokens, refreshTokens, nil)
	out, err := refreshUC.Execute(context.Background(), application.RefreshTokenInput{
		RefreshToken: loginOut.RefreshToken,
	})
	if err != nil {
		t.Fatalf("refresh error = %v", err)
	}
	if out.AccessToken == "" {
		t.Fatal("expected new access token")
	}
	if out.RefreshToken == "" {
		t.Fatal("expected rotated refresh token")
	}
	if out.RefreshToken == loginOut.RefreshToken {
		t.Fatal("expected refresh token to rotate")
	}

	if _, err := refreshUC.Execute(context.Background(), application.RefreshTokenInput{
		RefreshToken: loginOut.RefreshToken,
	}); err == nil {
		t.Fatal("expected old refresh token to be revoked")
	}

	if _, err := refreshUC.Execute(context.Background(), application.RefreshTokenInput{
		RefreshToken: out.RefreshToken,
	}); err != nil {
		t.Fatalf("new refresh token should work: %v", err)
	}
}

func TestRefreshToken_doesNotIssueAccessIfRotateFails(t *testing.T) {
	user := mustUser(t, "user-1", "rotate-fail@example.com")
	issuer := &recordingAccessIssuer{token: "access"}
	refresh := &recordingRefreshManager{userID: user.ID, rotateErr: errors.New("db down")}
	uc := application.NewRefreshToken(staticUserRepo{user: user}, issuer, refresh, nil)

	_, err := uc.Execute(context.Background(), application.RefreshTokenInput{RefreshToken: "old-refresh"})
	if err == nil {
		t.Fatal("expected rotate error")
	}
	if issuer.calls != 0 {
		t.Fatalf("Issue calls = %d, want 0", issuer.calls)
	}
	if refresh.rotateCalls != 1 {
		t.Fatalf("Rotate calls = %d, want 1", refresh.rotateCalls)
	}
}

func TestRefreshToken_rotatesBeforeIssuingAccess(t *testing.T) {
	user := mustUser(t, "user-1", "rotate-ok@example.com")
	var steps []string
	issuer := &recordingAccessIssuer{token: "access", onIssue: func() { steps = append(steps, "issue") }}
	refresh := &recordingRefreshManager{
		userID:   user.ID,
		newToken: "new-refresh",
		onRotate: func() { steps = append(steps, "rotate") },
	}
	uc := application.NewRefreshToken(staticUserRepo{user: user}, issuer, refresh, nil)

	out, err := uc.Execute(context.Background(), application.RefreshTokenInput{RefreshToken: "old-refresh"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.AccessToken != "access" || out.RefreshToken != "new-refresh" {
		t.Fatalf("output = %+v", out)
	}
	if len(steps) != 2 || steps[0] != "rotate" || steps[1] != "issue" {
		t.Fatalf("steps = %v, want [rotate issue]", steps)
	}
}

type staticUserRepo struct {
	user domain.User
}

func (r staticUserRepo) Create(context.Context, domain.User) error { return nil }

func (r staticUserRepo) GetByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, domain.ErrNotFound
}

func (r staticUserRepo) GetByID(_ context.Context, id string) (domain.User, error) {
	if id != r.user.ID {
		return domain.User{}, domain.ErrNotFound
	}
	return r.user, nil
}

type recordingAccessIssuer struct {
	token   string
	err     error
	calls   int
	onIssue func()
}

func (i *recordingAccessIssuer) Issue(string, []string) (string, error) {
	i.calls++
	if i.onIssue != nil {
		i.onIssue()
	}
	if i.err != nil {
		return "", i.err
	}
	return i.token, nil
}

type recordingRefreshManager struct {
	userID      string
	newToken    string
	rotateErr   error
	rotateCalls int
	onRotate    func()
}

func (m *recordingRefreshManager) Issue(context.Context, string) (string, error) {
	return "", nil
}

func (m *recordingRefreshManager) Validate(context.Context, string) (string, error) {
	return m.userID, nil
}

func (m *recordingRefreshManager) Revoke(context.Context, string) error { return nil }

func (m *recordingRefreshManager) Rotate(context.Context, string, string) (string, error) {
	m.rotateCalls++
	if m.onRotate != nil {
		m.onRotate()
	}
	if m.rotateErr != nil {
		return "", m.rotateErr
	}
	return m.newToken, nil
}

func mustUser(t *testing.T, id, email string) domain.User {
	t.Helper()
	user, err := domain.NewUser(id, email, "hash", []roles.Role{roles.RoleUser})
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	return user
}

func TestRefreshToken_usesLiveRolesFromRepository(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := sessionvalidation.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, nil)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "roles@example.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	user, err := repo.GetByEmail(context.Background(), "roles@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	user.Roles = append(user.Roles, roles.RoleTrader)
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute), nil)
	loginOut, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "roles@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("login error = %v", err)
	}

	refreshUC := application.NewRefreshToken(repo, accessTokens, refreshTokens, nil)
	out, err := refreshUC.Execute(context.Background(), application.RefreshTokenInput{
		RefreshToken: loginOut.RefreshToken,
	})
	if err != nil {
		t.Fatalf("refresh error = %v", err)
	}

	claims, err := accessTokens.Validate(out.AccessToken)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(claims.Roles) != 2 {
		t.Fatalf("Roles = %v, want user and trader", claims.Roles)
	}
}

func TestRefreshToken_missingUserIsNotUnauthorized(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	uc := application.NewRefreshToken(lookupUserRepo{err: domain.ErrNotFound}, nopAccessIssuer{}, validatingRefresh{}, zap.New(core))

	_, err := uc.Execute(context.Background(), application.RefreshTokenInput{RefreshToken: "old-refresh"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	if errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("error = %v, must stay distinct from ErrUnauthorized", err)
	}

	entries := logs.FilterMessage("refresh token user not found").All()
	if len(entries) != 1 {
		t.Fatalf("not-found logs = %d, want 1", len(entries))
	}
	if got, _ := entries[0].ContextMap()["user_id"].(string); got != "user-1" {
		t.Fatalf("user_id = %q, want user-1", got)
	}
}

func TestRefreshToken_databaseErrorStaysDatabaseError(t *testing.T) {
	dbErr := errors.New("connection reset")
	core, logs := observer.New(zapcore.ErrorLevel)
	uc := application.NewRefreshToken(lookupUserRepo{err: dbErr}, nopAccessIssuer{}, validatingRefresh{}, zap.New(core))

	_, err := uc.Execute(context.Background(), application.RefreshTokenInput{RefreshToken: "old-refresh"})
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want connection reset", err)
	}
	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("error = %v, database failure must not look like auth", err)
	}

	entries := logs.FilterMessage("refresh token user lookup failed").All()
	if len(entries) != 1 {
		t.Fatalf("lookup failure logs = %d, want 1", len(entries))
	}
	if strings.Contains(fmt.Sprint(entries[0].ContextMap()), "old-refresh") {
		t.Fatal("refresh token must not appear in lookup failure log")
	}
}

type lookupUserRepo struct {
	err error
}

func (lookupUserRepo) Create(context.Context, domain.User) error { return nil }

func (lookupUserRepo) GetByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, domain.ErrNotFound
}

func (r lookupUserRepo) GetByID(context.Context, string) (domain.User, error) {
	return domain.User{}, r.err
}

type nopAccessIssuer struct{}

func (nopAccessIssuer) Issue(string, []string) (string, error) { return "", nil }

type validatingRefresh struct{}

func (validatingRefresh) Issue(context.Context, string) (string, error) { return "", nil }

func (validatingRefresh) Validate(context.Context, string) (string, error) { return "user-1", nil }

func (validatingRefresh) Revoke(context.Context, string) error { return nil }

func (validatingRefresh) Rotate(context.Context, string, string) (string, error) {
	return "new-refresh", nil
}

func TestGetUser_success(t *testing.T) {
	repo := memory.NewUserRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := sessionvalidation.NewRefreshTokenService(memory.NewRefreshTokenRepository(), 24*time.Hour)
	register := application.NewRegister(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, nil)
	registerOut, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "profile@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("register error = %v", err)
	}

	getUser := application.NewGetUser(repo)
	out, err := getUser.Execute(context.Background(), application.GetUserInput{UserID: registerOut.UserID})
	if err != nil {
		t.Fatalf("get user error = %v", err)
	}
	if out.Email != "profile@example.com" {
		t.Fatalf("email = %q", out.Email)
	}
}

func TestLogout_revokesRefreshToken(t *testing.T) {
	repo := memory.NewUserRepository()
	refreshRepo := memory.NewRefreshTokenRepository()
	accessTokens, err := sessionvalidation.NewTokenService("test-secret", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenService() error = %v", err)
	}
	refreshTokens := sessionvalidation.NewRefreshTokenService(refreshRepo, 24*time.Hour)

	register := application.NewRegister(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, nil)
	if _, err := register.Execute(context.Background(), application.RegisterInput{
		Email:    "logout@example.com",
		Password: "Password1!",
	}); err != nil {
		t.Fatalf("register error = %v", err)
	}

	login := application.NewLogin(repo, bcrypt.NewHasher(0), accessTokens, refreshTokens, ratelimit.NewLoginLimiter(10, time.Minute), nil)
	loginOut, err := login.Execute(context.Background(), application.LoginInput{
		Email:    "logout@example.com",
		Password: "Password1!",
	})
	if err != nil {
		t.Fatalf("login error = %v", err)
	}

	logout := application.NewLogout(refreshTokens, nil)
	if err := logout.Execute(context.Background(), application.LogoutInput{RefreshToken: loginOut.RefreshToken}); err != nil {
		t.Fatalf("logout error = %v", err)
	}

	refreshUC := application.NewRefreshToken(repo, accessTokens, refreshTokens, nil)
	if _, err := refreshUC.Execute(context.Background(), application.RefreshTokenInput{RefreshToken: loginOut.RefreshToken}); err == nil {
		t.Fatal("expected refresh to fail after logout")
	}
}
