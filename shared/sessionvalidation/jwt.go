package sessionvalidation

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken возвращается при невалидном или просроченном JWT.
	ErrInvalidToken = errors.New("invalid token")
)

// Claims — полезная нагрузка access token.
type Claims struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenService выпускает и проверяет JWT access token.
type TokenService struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenService создаёт сервис JWT с заданным секретом и TTL.
func NewTokenService(secret string, ttl time.Duration) (*TokenService, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("jwt secret is required")
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &TokenService{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

// Issue создаёт подписанный access token.
func (s *TokenService) Issue(userID string, roles []string) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("user_id is required")
	}

	now := time.Now()
	claims := Claims{
		UserID: userID,
		Roles:  append([]string(nil), roles...),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// Validate проверяет token и возвращает claims.
func (s *TokenService) Validate(tokenString string) (Claims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return Claims{}, ErrInvalidToken
	}

	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return Claims{}, ErrInvalidToken
	}
	return *claims, nil
}
