package domain

import (
	"context"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// AccountContext represents the context of an account (sandbox status, mode, etc.)
type AccountContext struct {
	AccountID      string
	OwnerAccountID *string
	AccountMode    constants.AccountMode
}

// AccountUserAccess represents a user's access to an account
type AccountUserAccess struct {
	AccountUserID string
	AccountID     string
	RoleID        *string
	RoleTypeCode  *string
	Permissions   map[string]bool
}

// AuthCoreClient is the interface for core-service operations needed by auth-service
type AuthCoreClient interface {
	// GetAccountContext returns whether an account is a sandbox and its mode
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)

	// GetUserAccountAccess returns the user's role/permissions for an account
	GetUserAccountAccess(ctx context.Context, userID, accountID string) (*AccountUserAccess, *apierror.APIError)

	// GetAccountRelationByUserID returns the relationship between accounts based on user
	GetAccountRelationByUserID(ctx context.Context, ownerAccountID, userID string) (*AuthAccountRelation, *apierror.APIError)

	// GetAccountRelationByAPIKeyID returns the relationship between accounts based on API key
	GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AuthAccountRelation, *apierror.APIError)

	// MarkAccountUserUsed marks an account user as recently used
	MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError

	// GetRolePermissions returns the permissions for a role
	GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)
}
