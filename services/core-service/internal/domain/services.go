package domain

import (
	"context"
	"time"

	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

type SandboxSvc interface {
	// CreateSandbox creates a sandbox account with the given name.
	//
	// Preconditions:
	//   - The caller must be authorized to create sandbox accounts.
	//
	// Side effects:
	//   - Persists a new owner account and its associated sandbox account.
	CreateSandbox(ctx context.Context, name string, mode constants.SandboxMode) (*SandboxAccount, *apierror.APIError)

	// GetSandboxAccountByOwner returns the sandbox account ID associated with the given owner account.
	GetSandboxAccountByOwner(ctx context.Context, ownerAccountID string) (string, *apierror.APIError)

	// ListSandboxAccounts returns a paginated list of sandbox accounts visible to the caller.
	//
	// Pagination:
	//   - If cursor is non-nil, results begin after the provided cursor.
	//   - limit controls the maximum number of results returned.
	ListSandboxAccounts(ctx context.Context, cursor *string, limit int32) (*ListSandboxAccountsResult, *apierror.APIError)

	// GetSandbox returns a single sandbox account by its type ID. The caller must
	// have read permission on the sandbox domain and the sandbox must belong to the
	// caller's target account.
	GetSandbox(ctx context.Context, sandboxTypeID string) (*SandboxAccount, *apierror.APIError)

	// DeleteSandbox deletes a sandbox account and its underlying account record.
	// At least one sandbox must remain per production account. Account-scoped data
	// is purged asynchronously via an outbox message.
	DeleteSandbox(ctx context.Context, sandboxTypeID string) *apierror.APIError
}

type UnitSvc interface {
	// ListUnits returns a paginated list of units visible to the caller's
	// account. Includes both account-specific and global (system) units.
	ListUnits(ctx context.Context, params ListUnitsParams) (*ListUnitsResult, *apierror.APIError)
}

type AccountSvc interface {
	// GetAccountContext returns contextual information for an account (including whether it is a sandbox).
	GetAccountContext(ctx context.Context, accountID string) (*AccountContext, *apierror.APIError)

	// GetUserAccountAccess returns the user's access to an account, including role and permissions.
	//
	// If the user has no relationship to the account, returns (nil, false, nil).
	GetUserAccountAccess(ctx context.Context, userID, accountID string) (*AccountUserAccess, bool, *apierror.APIError)

	// GetRolePermissions returns the permission map for the given role ID.
	GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, *apierror.APIError)

	// GetAccountRelationByUserID returns the relationship between the owner account and the account implied by the user.
	GetAccountRelationByUserID(ctx context.Context, ownerAccountID, userID string) (*AccountRelation, *apierror.APIError)

	// GetAccountRelationByAPIKeyID returns the relationship between the owner account and the account implied by the API key.
	GetAccountRelationByAPIKeyID(ctx context.Context, ownerAccountID string, apiKeyID int64) (*AccountRelation, *apierror.APIError)

	// MarkAccountUserUsed records that the account user was recently used.
	MarkAccountUserUsed(ctx context.Context, accountUserID string) *apierror.APIError

	// ListUserAccountAffiliations returns the accounts the user is affiliated with.
	//
	// Also returns, if available, the user's last used account ID.
	ListUserAccountAffiliations(ctx context.Context, userID string) ([]AccountAffiliation, *string, *apierror.APIError)

	// GetAdminRole returns the role ID used for administrative access.
	GetAdminRole(ctx context.Context) (string, *apierror.APIError)

	// UpdateAccountSubscription updates subscription fields on an account,
	// resolving the account_plan_id from the plan_code.
	UpdateAccountSubscription(ctx context.Context, accountID string, status *string, planCode string, stripeSubID *string, periodEnd *time.Time, stripeCustomerID *string) *apierror.APIError

	// ClearAccountStripeCustomer removes all Stripe-related fields from an account.
	ClearAccountStripeCustomer(ctx context.Context, accountID string) *apierror.APIError

	// GetAccountByStripeCustomerID resolves an account from a Stripe customer ID.
	GetAccountByStripeCustomerID(ctx context.Context, stripeCustomerID string) (accountID string, planCode string, err *apierror.APIError)

	// CompleteRegistration creates the production account, sandbox, owner roles,
	// account-user records, business address, and portal for a newly registered
	// user. Returns the new account ID and sandbox account ID.
	CompleteRegistration(ctx context.Context, input CompleteRegistrationInput) (*CompleteRegistrationOutput, *apierror.APIError)
}

// CompleteRegistrationInput carries the data needed to finalize a registration.
type CompleteRegistrationInput struct {
	UserID               string
	PlanCode             string
	StripeCustomerID     string
	StripeSubscriptionID string
	AccountData          RegistrationAccountData
	BusinessAddress      *RegistrationAddress
}

// RegistrationAccountData holds the business profile information collected
// during onboarding.
type RegistrationAccountData struct {
	AccountName string
}

// RegistrationAddress is a structured postal address collected during
// registration.
type RegistrationAddress struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

// CompleteRegistrationOutput holds the IDs of the newly created accounts.
type CompleteRegistrationOutput struct {
	AccountID string
	SandboxID string
}

// CreateAccountParams holds the parameters for creating a production account
// during registration.
type CreateAccountParams struct {
	ID                   string
	Name                 string
	PlanCode             string
	StripeCustomerID     string
	StripeSubscriptionID string
}
