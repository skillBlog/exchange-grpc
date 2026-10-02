package application

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"

	"github.com/exchange-grpc/userservice/internal/domain"
)

const minPasswordLength = 8

// ValidateEmail проверяет формат email.
func ValidateEmail(email string) error {
	email = domain.NormalizeEmail(email)
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

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	switch {
	case !hasUpper:
		return fmt.Errorf("%w: password must contain at least one uppercase letter", domain.ErrInvalidArgument)
	case !hasLower:
		return fmt.Errorf("%w: password must contain at least one lowercase letter", domain.ErrInvalidArgument)
	case !hasDigit:
		return fmt.Errorf("%w: password must contain at least one digit", domain.ErrInvalidArgument)
	case !hasSpecial:
		return fmt.Errorf("%w: password must contain at least one special character", domain.ErrInvalidArgument)
	}

	return nil
}
