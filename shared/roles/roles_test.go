package roles_test

import (
	"testing"

	"github.com/exchange-grpc/shared/roles"
)

func TestNormalize_caseInsensitive(t *testing.T) {
	if got := roles.Normalize("Admin"); got != roles.RoleAdmin {
		t.Fatalf("Normalize(Admin) = %q, want admin", got)
	}
}

func TestParse_rejectsUnknownRole(t *testing.T) {
	if _, ok := roles.Parse("superuser"); ok {
		t.Fatal("expected unknown role to be rejected")
	}
}

func TestMatch_roleCaseInsensitive(t *testing.T) {
	if !roles.Match([]string{"trader", "admin"}, []string{"Admin"}) {
		t.Fatal("expected Admin to match admin in allowed roles")
	}
}

func TestNormalizeStrings_deduplicates(t *testing.T) {
	got := roles.NormalizeStrings([]string{"trader", "Trader", " admin "})
	want := []string{"trader", "admin"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
