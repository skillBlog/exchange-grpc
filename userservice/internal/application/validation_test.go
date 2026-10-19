package application_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/exchange-grpc/userservice/internal/application"
	"github.com/exchange-grpc/userservice/internal/domain"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "valid", password: "Password1!", wantErr: false},
		{name: "too short", password: "Pw1!", wantErr: true},
		{name: "no uppercase", password: "password1!", wantErr: true},
		{name: "no lowercase", password: "PASSWORD1!", wantErr: true},
		{name: "no digit", password: "Password!", wantErr: true},
		{name: "no special", password: "Password1", wantErr: true},
		{name: "digits only", password: "12345678", wantErr: true},
	}

	var clientMessage string
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := application.ValidatePassword(tc.password)
			if tc.wantErr {
				if !errors.Is(err, domain.ErrInvalidArgument) {
					t.Fatalf("error = %v, want ErrInvalidArgument", err)
				}
				if clientMessage == "" {
					clientMessage = err.Error()
				}
				if err.Error() != clientMessage {
					t.Fatalf("error = %q, want the same client message %q", err.Error(), clientMessage)
				}
				assertPasswordErrorHidesRule(t, err)
				return
			}
			if err != nil {
				t.Fatalf("ValidatePassword() error = %v", err)
			}
		})
	}
}

func assertPasswordErrorHidesRule(t *testing.T, err error) {
	t.Helper()
	msg := err.Error()
	for _, hint := range []string{"uppercase", "lowercase", "digit", "special", "characters", "at least"} {
		if strings.Contains(msg, hint) {
			t.Fatalf("client error %q reveals password rule %q", msg, hint)
		}
	}
}
