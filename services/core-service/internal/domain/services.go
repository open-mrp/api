package domain

import (
	"context"

	apierror "github.com/augno/api/shared/errors"
)

type AccountSvc interface {
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)
	GetUserAccountAccess(ctx context.Context, userID, accountID string) (*AccountUserAccess, *apierror.APIError)
	GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)
	GetAccountRelationByUserID(ctx context.Context, ownerAccountID, userID string) (*AccountRelation, *apierror.APIError)
	GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AccountRelation, *apierror.APIError)
	MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError
	ListUserAccountAffiliations(ctx context.Context, userID string) ([]AccountAffiliation, *string, *apierror.APIError)
}
