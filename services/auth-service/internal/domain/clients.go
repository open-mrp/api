package domain

import (
	"context"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// AccountContext represents the context of an account (sandbox status, mode, etc.)
type AccountContext struct {
	AccountID          string
	OwnerAccountID     *string
	AccountMode        constants.AccountMode
	SubscriptionStatus *string
}

// AccountUserAccess represents a user's access to an account
type AccountUserAccess struct {
	AccountUserID string
	AccountID     string
	RoleID        *string
	RoleType      *string
	RoleName      *string
	Permissions   map[string]bool
}

// AuthCoreClient is the interface for core-service operations needed by auth-service
type AuthCoreClient interface {
	// GetAccountContext returns whether an account is a sandbox and its mode
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)

	// GetUserAccountAccess returns the user's role/permissions for an account
	GetUserAccountAccess(ctx context.Context, userID, accountID string) (*AccountUserAccess, bool, *apierror.APIError)

	// GetAccountRelationByUserID returns the relationship between accounts based on user. actorAccountID is required to unlock owner-side matches (the relation's owner_account_id must equal it); pass "" when no actor account has been validated to skip owner-side.
	GetAccountRelationByUserID(ctx context.Context, targetAccountID, actorAccountID, userID string) (*AuthAccountRelation, bool, *apierror.APIError)

	// GetAccountRelationByAPIKeyID returns the relationship between accounts based on API key
	GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AuthAccountRelation, bool, *apierror.APIError)

	// MarkAccountUserUsed marks an account user as recently used
	MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError

	// GetRolePermissions returns the permissions for a role
	GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)

	// GetSandboxAccountByOwner returns the sandbox account ID for a given owner account
	GetSandboxAccountByOwner(ctx context.Context, ownerAccountID string) (string, *apierror.APIError)

	// GetAdminRole returns the admin role ID
	GetAdminRole(ctx context.Context) (string, *apierror.APIError)
}
