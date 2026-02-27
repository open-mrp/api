package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type AccountRepo interface {
	Create(ctx context.Context, id, name string, accountTypeCode AccountType, planCode constants.PlanCode) *apierror.APIError
	GetPlanCode(ctx context.Context, id string) (constants.PlanCode, *apierror.APIError)
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)
	Delete(ctx context.Context, id string) *apierror.APIError
	GetPlanTypeIDByCode(ctx context.Context, planCode string) (string, *apierror.APIError)
	UpdateSubscription(ctx context.Context, accountID string, status *string, planCode string, accountPlanID *string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string) *apierror.APIError
	ClearStripeCustomer(ctx context.Context, accountID string) *apierror.APIError
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (accountID string, planCode string, err *apierror.APIError)
	GetSandboxLimit(ctx context.Context, accountID string) (*int32, *apierror.APIError)
	GetSeatLimitByPlanCode(ctx context.Context, planCode string) (*int32, *apierror.APIError)
	CountNonSandboxByPlanCode(ctx context.Context, planCode string) (int64, *apierror.APIError)
}

type AccountUserRepo interface {
	FindByAccountAndUserID(ctx context.Context, userID, accountID string) (*AccountUser, *apierror.APIError)
	FindAffiliationsByUserID(ctx context.Context, userID string) ([]AccountAffiliation, *apierror.APIError)
	FindLastUsedAccountID(ctx context.Context, userID string) (string, *apierror.APIError)
	UpdateLastUsedAt(ctx context.Context, accountUserID string, lastUsedAt time.Time) *apierror.APIError
	GetAdminRoleID(ctx context.Context) (string, *apierror.APIError)
	DeactivateExcept(ctx context.Context, accountID, keepUserID string, limit int32) (int64, *apierror.APIError)
	EnsureActive(ctx context.Context, accountID, userID string) *apierror.APIError
	FindAdminUserIDByAccountID(ctx context.Context, accountID string) (string, *apierror.APIError)
	CountActive(ctx context.Context, accountID string) (int64, *apierror.APIError)
	ReactivateUsers(ctx context.Context, accountID string, limit int32) (int64, *apierror.APIError)
}

type SandboxAccountRepo interface {
	FindFirstByOwnerAccountID(ctx context.Context, ownerAccountID string) (string, *apierror.APIError)
	FindByTypeID(ctx context.Context, typeID string) (*SandboxAccount, *apierror.APIError)
	List(ctx context.Context, ownerAccountID string, cursor *string, limit int32) (*ListSandboxAccountsResult, *apierror.APIError)
	Create(ctx context.Context, typeID, ownerAccountID, accountID string) *apierror.APIError
	CountByOwnerAccountID(ctx context.Context, ownerAccountID string) (int64, *apierror.APIError)
	DeleteByID(ctx context.Context, id int64) *apierror.APIError
}

type AccountRelationRepo interface {
	FindByOwnerAccountAndUserID(ctx context.Context, ownerAccountID, userID string) (*AccountRelation, *apierror.APIError)
	FindByOwnerAccountAndAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AccountRelation, *apierror.APIError)
}

type RolePermissionRepo interface {
	FindByRoleID(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)
}

// RegistrationRepo handles the multi-step account creation during
// registration: account record, owner role with all permissions,
// account-user join, business address, and account portal.
type RegistrationRepo interface {
	// CreateAccountForRegistration creates a production account with Stripe
	// billing fields populated.
	CreateAccountForRegistration(ctx context.Context, params CreateAccountParams) *apierror.APIError

	// CreateAccountUser creates an account-user join record linking the
	// user to the account with the given role.
	CreateAccountUser(ctx context.Context, accountID, userID, roleID string) *apierror.APIError

	// CreateBusinessAddress creates a geolocation, address, and
	// account-address chain and sets the address as the account's default
	// billing and shipping address.
	CreateBusinessAddress(ctx context.Context, accountID, accountName string, address RegistrationAddress) *apierror.APIError

	// CreateAccountPortal creates a portal record with a slug derived from
	// the account ID.
	CreateAccountPortal(ctx context.Context, accountID string) *apierror.APIError
}

type UnitRepo interface {
	List(ctx context.Context, params ListUnitsParams) (*ListUnitsResult, *apierror.APIError)
}

type IdempotencyKeyRepo interface {
	GetByScopeHash(ctx context.Context, scopeHash string) (*IdempotencyKey, *apierror.APIError)
	Create(ctx context.Context, key *IdempotencyKey) (*IdempotencyKey, *apierror.APIError)
	AdvanceRecoveryPoint(ctx context.Context, typeID string, recoveryPoint RecoveryPoint) *apierror.APIError
	GetRecoveryPoint(ctx context.Context, typeID string) (RecoveryPoint, *apierror.APIError)
	SetResponse(ctx context.Context, typeID string, code int, body json.RawMessage, recoveryPoint RecoveryPoint) *apierror.APIError
}
