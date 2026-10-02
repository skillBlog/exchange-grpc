package domain_test

import (
	"testing"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/google/uuid"
)

func TestNewOrderID_isVersion7(t *testing.T) {
	id, err := uuid.Parse(domain.NewOrderID())
	if err != nil {
		t.Fatalf("NewOrderID() parse error = %v", err)
	}
	if id.Version() != 7 {
		t.Fatalf("version = %d, want 7", id.Version())
	}
}
