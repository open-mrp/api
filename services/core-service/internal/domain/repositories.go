package domain

import (
	"context"
	"encoding/json"
	"time"

	apierror "github.com/augno/api/shared/errors"
)

type AccountRepo interface {
	Create(ctx context.Context, id, name string, accountTypeCode AccountType, planCode string) *apierror.APIError
	GetPlanCode(ctx context.Context, id string) (string, *apierror.APIError)
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)
	Delete(ctx context.Context, id string) *apierror.APIError
}

type AccountUserRepo interface {
	FindByAccountAndUserID(ctx context.Context, userID, accountID string) (*AccountUser, *apierror.APIError)
	FindAffiliationsByUserID(ctx context.Context, userID string) ([]AccountAffiliation, *apierror.APIError)
	FindLastUsedAccountID(ctx context.Context, userID string) (string, *apierror.APIError)
	UpdateLastUsedAt(ctx context.Context, accountUserID string, lastUsedAt time.Time) *apierror.APIError
}

type AccountRelationRepo interface {
	FindByOwnerAccountAndUserID(ctx context.Context, ownerAccountID, userID string) (*AccountRelation, *apierror.APIError)
	FindByOwnerAccountAndAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AccountRelation, *apierror.APIError)
}

type RolePermissionRepo interface {
	FindByRoleID(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)
}

type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (RecoveryPoint, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
