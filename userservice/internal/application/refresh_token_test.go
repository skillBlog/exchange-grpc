package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/shared/sessionvalidation"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/infrastructure/bcrypt"
	"github.com/exchange-grpc/userservice/internal/infrastructure/memory"
	"github.com/exchange-grpc/userservice/internal/infrastructure/ratelimit"
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
	if err := repo.Save(context.Background(), user); err != nil {
		t.Fatalf("Save() error = %v", err)
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
