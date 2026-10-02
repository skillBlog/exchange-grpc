package application

import (
	"errors"
	"testing"

	gocrypto "golang.org/x/crypto/bcrypt"
)

func TestDummyPasswordHashIsComparable(t *testing.T) {
	err := gocrypto.CompareHashAndPassword([]byte(dummyPasswordHash), []byte("not-the-dummy"))
	if err == nil {
		t.Fatal("dummy hash must not match an arbitrary password")
	}
	if errors.Is(err, gocrypto.ErrHashTooShort) {
		t.Fatal("dummyPasswordHash is not a valid bcrypt hash")
	}
}
