package grpcserver

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	commonv1 "github.com/exchange-grpc/proto/pb/common/v1"
	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRolesToProto_skipsUnknownAndEmpty(t *testing.T) {
	got := rolesToProto([]string{"user", "auditor", "trader", ""})
	want := []commonv1.Role{
		commonv1.Role_ROLE_USER,
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

func TestToGRPCError_passwordAndRefreshLookupAreDistinct(t *testing.T) {
	short := toGRPCError(application.ValidatePassword("short"))
	weak := toGRPCError(application.ValidatePassword("password1!"))
	shortSt, ok := status.FromError(short)
	if !ok {
		t.Fatal("short password error is not a status")
	}
	weakSt, ok := status.FromError(weak)
	if !ok {
		t.Fatal("weak password error is not a status")
	}
	if shortSt.Code() != codes.InvalidArgument || weakSt.Code() != codes.InvalidArgument {
		t.Fatalf("codes = %s and %s, want InvalidArgument", shortSt.Code(), weakSt.Code())
	}
	if shortSt.Message() != weakSt.Message() {
		t.Fatalf("messages = %q and %q, want one password error", shortSt.Message(), weakSt.Message())
	}
	if strings.Contains(shortSt.Message(), "uppercase") || strings.Contains(weakSt.Message(), "digit") {
		t.Fatalf("status messages reveal password rules: %q, %q", shortSt.Message(), weakSt.Message())
	}

	missing, _ := status.FromError(toGRPCError(fmt.Errorf("get user: %w", domain.ErrNotFound)))
	dbErr, _ := status.FromError(toGRPCError(fmt.Errorf("get user: %w", errors.New("connection reset"))))
	if missing.Code() != codes.NotFound {
		t.Fatalf("missing user code = %s, want NotFound", missing.Code())
	}
	if dbErr.Code() != codes.Internal {
		t.Fatalf("database code = %s, want Internal", dbErr.Code())
	}
	if strings.Contains(dbErr.Message(), "connection reset") {
		t.Fatalf("database status %q leaks the driver error", dbErr.Message())
	}
}

func TestGetUserOutputToResponse_omitsUnknownRole(t *testing.T) {
	resp := Mapper{}.GetUserOutputToResponse(application.GetUserOutput{
		UserID: "user-1",
		Email:  "a@b.com",
		Roles:  []string{"auditor"},
	})
	if len(resp.GetRoles()) != 0 {
		t.Fatalf("roles = %v, want none", resp.GetRoles())
	}
}
