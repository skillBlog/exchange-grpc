package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	sharedgrpc "github.com/exchange-grpc/shared/grpc"
	"github.com/exchange-grpc/shared/logger"
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

	if uc.limiter != nil {
		if err = uc.limiter.Allow(ctx, email); err != nil {
			return LoginOutput{}, err
		}
	}

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) || errors.Is(err, domain.ErrNotFound) {
			uc.compareDummy(password)
			uc.logFailedLogin(ctx, email, input.ClientAddr, loginFailReasonUserNotFound)
			return LoginOutput{}, domain.ErrUnauthorized
		}
		return LoginOutput{}, err
	}

	span.SetAttributes(tracing.Attr("user_id", user.ID))

	if err = uc.hasher.Compare(user.PasswordHash, password); err != nil {
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

	return LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (uc *Login) compareDummy(password string) {
	hash := uc.dummyPasswordHash()
	if hash == "" {
		return
	}
	_ = uc.hasher.Compare(hash, password)
}

func (uc *Login) dummyPasswordHash() string {
	dummyHashOnce.Do(func() {
		hash, err := uc.hasher.Hash(timingDummyPassword)
		if err != nil {
			return
		}
		dummyHash = hash
	})
	return dummyHash
}

func (uc *Login) logFailedLogin(ctx context.Context, email, clientAddr, reason string) {
	fields := []zap.Field{
		zap.String("email", email),
		zap.String("reason", reason),
	}
	if requestID := sharedgrpc.RequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if clientAddr != "" {
		fields = append(fields, zap.String("client_addr", clientAddr))
	}
	logger.WithTrace(ctx, uc.log).Warn("login failed", fields...)
}
