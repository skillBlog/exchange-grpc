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

const genericPasswordMessage = "password does not meet requirements"

// ValidatePassword проверяет сложность пароля при регистрации.
// Наружу уходит одна ошибка: конкретное правило остаётся только во внутреннем логе.
func ValidatePassword(password string) error {
	if passwordRejectReason(password) == "" {
		return nil
	}
	return fmt.Errorf("%w: %s", domain.ErrInvalidArgument, genericPasswordMessage)
}

// passwordRejectReason возвращает код нарушенного правила или пустую строку.
// Код пишется в лог и не должен попадать в ответ клиенту.
func passwordRejectReason(password string) string {
	password = strings.TrimSpace(password)
	if len(password) < minPasswordLength {
		return "too_short"
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
		return "missing_uppercase"
	case !hasLower:
		return "missing_lowercase"
	case !hasDigit:
		return "missing_digit"
	case !hasSpecial:
		return "missing_special"
	default:
		return ""
	}
}
