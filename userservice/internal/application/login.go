package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/tracing"
	"github.com/exchange-grpc/userservice/internal/domain"
	"go.uber.org/zap"
)

const (
	loginFailReasonUserNotFound = "user-not-found"
	loginFailReasonBadPassword  = "bad-password"
	timingDummyPassword         = "timing-dummy"
)

var (
	dummyHashOnce sync.Once
	dummyHash     string
)

// LoginInput — параметры входа пользователя.
type LoginInput struct {
	Email      string
	Password   string
	ClientAddr string
}

// LoginOutput — результат успешного входа.
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
}

// Login аутентифицирует пользователя и выпускает токены.
type Login struct {
	users         domain.UserRepository
	hasher        PasswordHasher
	accessTokens  AccessTokenIssuer
	refreshTokens RefreshTokenManager
	limiter       LoginRateLimiter
	log           *zap.Logger
}

// NewLogin создаёт use case Login.
func NewLogin(
	users domain.UserRepository,
	hasher PasswordHasher,
	accessTokens AccessTokenIssuer,
	refreshTokens RefreshTokenManager,
	limiter LoginRateLimiter,
	log *zap.Logger,
) *Login {
	if log == nil {
		log = zap.NewNop()
	}
	return &Login{
		users:         users,
		hasher:        hasher,
		accessTokens:  accessTokens,
		refreshTokens: refreshTokens,
		limiter:       limiter,
		log:           log,
	}
}

// Execute проверяет пароль и возвращает access/refresh token.
func (uc *Login) Execute(ctx context.Context, input LoginInput) (out LoginOutput, err error) {
	ctx, span := tracing.Start(ctx, "user.Login")
	defer tracing.End(span, &err)

	email := NormalizeEmail(input.Email)
	password := strings.TrimSpace(input.Password)
	if err = ValidateEmail(email); err != nil {
		return LoginOutput{}, err
	}
	if password == "" {
		return LoginOutput{}, fmt.Errorf("%w: password is required", domain.ErrInvalidArgument)
	}

	if err = checkLoginRateLimit(ctx, uc.limiter, email); err != nil {
		return LoginOutput{}, fmt.Errorf("login rate limit: %w", err)
	}

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		uc.compareDummy(ctx, password)
		if errors.Is(err, domain.ErrUnauthorized) || errors.Is(err, domain.ErrNotFound) {
			uc.logFailedLogin(ctx, email, input.ClientAddr, loginFailReasonUserNotFound)
			return LoginOutput{}, domain.ErrUnauthorized
		}
		return LoginOutput{}, fmt.Errorf("get user: %w", err)
	}

	span.SetAttributes(tracing.Attr("user_id", user.ID))

	if err = comparePassword(ctx, uc.hasher, user.PasswordHash, password); err != nil {
		uc.logFailedLogin(ctx, email, input.ClientAddr, loginFailReasonBadPassword)
		return LoginOutput{}, domain.ErrUnauthorized
	}

	accessToken, err := uc.accessTokens.Issue(user.ID, user.RoleStrings())
	if err != nil {
		return LoginOutput{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := uc.refreshTokens.Issue(ctx, user.ID)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("issue refresh token: %w", err)
	}

	var extra []zap.Field
	if input.ClientAddr != "" {
		extra = append(extra, zap.String("client_addr", input.ClientAddr))
	}
	logAudit(ctx, uc.log, "user logged in", user.ID, extra...)

	return LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (uc *Login) compareDummy(ctx context.Context, password string) {
	hash := uc.dummyPasswordHash(ctx)
	if hash == "" {
		return
	}
	_ = comparePassword(ctx, uc.hasher, hash, password)
}

func (uc *Login) dummyPasswordHash(ctx context.Context) string {
	dummyHashOnce.Do(func() {
		hash, err := hashPassword(ctx, uc.hasher, timingDummyPassword)
		if err != nil {
			return
		}
		dummyHash = hash
	})
	return dummyHash
}

func (uc *Login) logFailedLogin(ctx context.Context, email, clientAddr, reason string) {
	fields := []zap.Field{
		zap.String("email_hash", hashEmail(email)),
		zap.String("reason", reason),
	}
	if clientAddr != "" {
		fields = append(fields, zap.String("client_addr", clientAddr))
	}
	sharedgrpc.LogWarn(ctx, uc.log, "login failed", "", fields...)
}
