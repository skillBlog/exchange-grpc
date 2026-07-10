package domain_test

import (
	"errors"
	"testing"

	sharederrors "github.com/exchange-grpc/shared/errors"
	"github.com/exchange-grpc/userservice/internal/domain"
)

func TestDomainErrorsUseSharedSentinels(t *testing.T) {
	if !errors.Is(domain.ErrNotFound, sharederrors.ErrNotFound) {
		t.Fatal("domain.ErrNotFound should alias shared error")
	}
	if !errors.Is(domain.ErrRateLimited, sharederrors.ErrRateLimited) {
		t.Fatal("domain.ErrRateLimited should alias shared error")
	}
}
