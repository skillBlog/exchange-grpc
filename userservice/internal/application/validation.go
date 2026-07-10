package application

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"

	"github.com/exchange-grpc/userservice/internal/domain"
)

const minPasswordLength = 8

// NormalizeEmail приводит email к каноническому виду.
func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

// ValidateEmail проверяет формат email.
func ValidateEmail(email string) error {
	email = NormalizeEmail(email)
	if email == "" {
		return fmt.Errorf("%w: email is required", domain.ErrInvalidArgument)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("%w: invalid email format", domain.ErrInvalidArgument)
	}
	return nil
}

// ValidatePassword проверяет сложность пароля при регистрации.
func ValidatePassword(password string) error {
	password = strings.TrimSpace(password)
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: password must be at least %d characters", domain.ErrInvalidArgument, minPasswordLength)
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("%w: password must contain letters and digits", domain.ErrInvalidArgument)
	}
	return nil
}
