package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

type AccountSvc interface {
	// GetAccountContext returns the context of an account including whether it's a sandbox.
	//
	//  1. Calls the account repository to get the account context.
	//  2. Returns the account context.
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)

	// GetUserAccountAccess returns the user's access to an account including their role and permissions.
	//
	//  1. Calls the account user repository to get the account user.
	//  2. Returns the account user access.
	GetUserAccountAccess(ctx context.Context, userID, accountID string) (*AccountUserAccess, *apierror.APIError)

	// GetRolePermissions returns the permissions for a role.
	//
	//  1. Calls the role permission repository to get the role permissions.
	//  2. Returns the role permissions.
	GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)

	// GetAccountRelationByUserID returns the relationship between accounts based on user.
	//
	//  1. Calls the account relation repository to get the account relation.
	//  2. Returns the account relation.
	GetAccountRelationByUserID(ctx context.Context, ownerAccountID, userID string) (*AccountRelation, *apierror.APIError)

	// GetAccountRelationByAPIKeyID returns the relationship between accounts based on API key.
	//
	//  1. Calls the account relation repository to get the account relation.
	//  2. Returns the account relation.
	GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AccountRelation, *apierror.APIError)

	// MarkAccountUserUsed marks an account user as recently used.
	//
	//  1. Calls the account user repository to update the last used at.
	//  2. Returns an error if the account user repository returns an error.
	MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError

	// ListUserAccountAffiliations returns the accounts a user is affiliated with.
	//
	//  1. Calls the account user repository to get the account user affiliations.
	//  2. Calls the account user repository to get the last used account ID.
	//  3. Returns the account user affiliations and the last used account ID.
	ListUserAccountAffiliations(ctx context.Context, userID string) ([]AccountAffiliation, *string, *apierror.APIError)

	// GetSandboxAccountByOwner returns the sandbox account ID for a given owner account.
	GetSandboxAccountByOwner(ctx context.Context, ownerAccountID string) (string, *apierror.APIError)

	// GetAdminRole returns the admin role ID.
	GetAdminRole(ctx context.Context) (string, *apierror.APIError)
}
