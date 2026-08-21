package domain

import (
	"context"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
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

// PlanInfo holds the pricing plan data returned by the billing service.
type PlanInfo struct {
	TypeID        string
	Name          string
	PlanTypeCode  string
	PricePerSeat  float64
	PricePerMonth *float64
	SeatMinimum   *int
}

// AuthBillingClient is the interface for billing-service operations needed by auth-service.
type AuthBillingClient interface {
	GetPlanByCode(ctx context.Context, planCode string) (*PlanInfo, *apierror.APIError)
	CreateCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*StripeCustomer, *apierror.APIError)
	SetupBillingProfile(ctx context.Context, accountID string) (*BillingProfileResult, *apierror.APIError)
	SubscribeToPricingPlan(ctx context.Context, stripeCustomerID, planCode string) *apierror.APIError
	CreateSetupIntent(ctx context.Context, customerID, idempotencyKey string) (*SetupIntentResult, *apierror.APIError)
	GetSetupIntentStatus(ctx context.Context, setupIntentID string) (*SetupIntentResult, *apierror.APIError)
	ValidateStripePricingPlan(ctx context.Context, planCode string) *apierror.APIError
	Close() error
	WaitForReady(ctx context.Context) error
}

// SetupIntentResult holds the result of a Setup Intent operation from billing-service.
type SetupIntentResult struct {
	SetupIntentID   string
	ClientSecret    string // #nosec G117 -- Stripe ephemeral client secret
	Status          string
	PaymentMethodID *string
	PublishableKey  string
}

// StripeCustomer represents a Stripe customer created during registration.
type StripeCustomer struct {
	ID string
}

// BillingProfileResult holds the IDs of the created billing profile and cadence.
type BillingProfileResult struct {
	ProfileID string
	CadenceID string
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

	// CompleteRegistration creates the production account, sandbox, roles, and permissions via core-service.
	CompleteRegistration(ctx context.Context, input CompleteAccountRegistrationInput) (*CompleteRegistrationOutput, *apierror.APIError)
}
