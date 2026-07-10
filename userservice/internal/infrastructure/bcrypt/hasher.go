package bcrypt

import (
	"golang.org/x/crypto/bcrypt"
)

const defaultCost = 12

// Hasher реализует application.PasswordHasher через bcrypt.
type Hasher struct {
	cost int
}

// NewHasher создаёт bcrypt-hasher.
func NewHasher() *Hasher {
	return &Hasher{cost: defaultCost}
}

// Hash возвращает bcrypt-хеш пароля.
func (h *Hasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare сверяет пароль с хешем.
func (h *Hasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
