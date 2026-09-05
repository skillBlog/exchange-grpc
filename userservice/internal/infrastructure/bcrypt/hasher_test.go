package bcrypt

import (
	"testing"

	gocrypto "golang.org/x/crypto/bcrypt"
)

func TestNewHasher_clampsCost(t *testing.T) {
	if got := NewHasher(0).Cost(); got != defaultCost {
		t.Fatalf("cost 0 = %d, want %d", got, defaultCost)
	}
	if got := NewHasher(gocrypto.MinCost - 1).Cost(); got != defaultCost {
		t.Fatalf("below min = %d, want %d", got, defaultCost)
	}
	if got := NewHasher(gocrypto.MaxCost + 1).Cost(); got != defaultCost {
		t.Fatalf("above max = %d, want %d", got, defaultCost)
	}
	if got := NewHasher(gocrypto.MinCost).Cost(); got != gocrypto.MinCost {
		t.Fatalf("min cost = %d, want %d", got, gocrypto.MinCost)
	}
	if got := NewHasher(defaultCost).Cost(); got != defaultCost {
		t.Fatalf("default = %d, want %d", got, defaultCost)
	}
}
