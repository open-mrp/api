package types

import (
	"testing"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func TestCheckNotSandboxMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		identity     *Identity
		expectedCode apierror.ErrorCode
		expectNil    bool
	}{
		{
			name: "rejects sandbox mode",
			identity: &Identity{
				Type:        IdentityActorTypeUser,
				AccountMode: constants.AccountModeSandbox,
				Actor:       &IdentityActor{ID: "usr_123"},
			},
			expectedCode: apierror.ErrorCodeInsufficientPerms,
		},
		{
			name: "allows production mode",
			identity: &Identity{
				Type:        IdentityActorTypeUser,
				AccountMode: constants.AccountModeProduction,
				Actor:       &IdentityActor{ID: "usr_123"},
			},
			expectNil: true,
		},
		{
			name: "rejects unauthenticated",
			identity: &Identity{
				Type: IdentityActorTypeUnauthenticated,
			},
			expectedCode: apierror.ErrorCodeInvalidCredentials,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.identity.CheckNotSandboxMode()
			if tt.expectNil {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Code != tt.expectedCode {
				t.Fatalf("expected error code %q, got %q", tt.expectedCode, err.Code)
			}
		})
	}
}
