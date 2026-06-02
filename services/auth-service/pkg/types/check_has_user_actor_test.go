package types

import (
	"testing"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func TestCheckHasUserActor(t *testing.T) {
	t.Parallel()

	accountID := "acct_123"

	tests := []struct {
		name         string
		identity     *Identity
		expectedCode apierror.ErrorCode
		expectNil    bool
	}{
		{
			// Regression: a freshly authenticated user hits /v1/identity/me/tenancy
			// before selecting an account, so the identity has no actor account.
			// CheckIsUser would 403 here; CheckHasUserActor must allow it.
			name: "allows authenticated user without an actor account",
			identity: &Identity{
				Type:        IdentityActorTypeUser,
				AccountMode: constants.AccountModeProduction,
				Actor:       &IdentityActor{ID: "usr_123", AccountID: nil},
			},
			expectNil: true,
		},
		{
			name: "allows authenticated user with an actor account",
			identity: &Identity{
				Type:        IdentityActorTypeUser,
				AccountMode: constants.AccountModeProduction,
				Actor:       &IdentityActor{ID: "usr_123", AccountID: &accountID},
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
		{
			name: "rejects non-user actor (api key)",
			identity: &Identity{
				Type:        IdentityActorTypeAPIKey,
				AccountMode: constants.AccountModeProduction,
				Actor:       &IdentityActor{ID: "key_123", AccountID: &accountID},
			},
			expectedCode: apierror.ErrorCodeInsufficientPerms,
		},
		{
			name: "rejects user with no actor",
			identity: &Identity{
				Type:        IdentityActorTypeUser,
				AccountMode: constants.AccountModeProduction,
				Actor:       nil,
			},
			expectedCode: apierror.ErrorCodeInsufficientPerms,
		},
		{
			name: "rejects user with empty actor id",
			identity: &Identity{
				Type:        IdentityActorTypeUser,
				AccountMode: constants.AccountModeProduction,
				Actor:       &IdentityActor{ID: ""},
			},
			expectedCode: apierror.ErrorCodeInsufficientPerms,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.identity.CheckHasUserActor()
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
