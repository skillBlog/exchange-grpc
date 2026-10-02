package grpcserver

import (
	"testing"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	"github.com/exchange-grpc/userservice/internal/application"
)

func TestRolesToProto_unknownBecomesUnspecified(t *testing.T) {
	got := rolesToProto([]string{"user", "auditor", "trader", ""})
	want := []commonv1.Role{
		commonv1.Role_ROLE_USER,
		commonv1.Role_ROLE_UNSPECIFIED,
		commonv1.Role_ROLE_TRADER,
	}
	if len(got) != len(want) {
		t.Fatalf("roles = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("roles = %v, want %v", got, want)
		}
	}
}

func TestGetUserOutputToResponse_keepsUnknownRole(t *testing.T) {
	resp := Mapper{}.GetUserOutputToResponse(application.GetUserOutput{
		UserID: "user-1",
		Email:  "a@b.com",
		Roles:  []string{"auditor"},
	})
	if len(resp.GetRoles()) != 1 || resp.GetRoles()[0] != commonv1.Role_ROLE_UNSPECIFIED {
		t.Fatalf("roles = %v, want [ROLE_UNSPECIFIED]", resp.GetRoles())
	}
}
