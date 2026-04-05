package apiresource

import (
	"github.com/augno/api/shared/constants"
)

// Tenancy represents the authenticated user's tenancy context.
type Tenancy struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=tenancy"`
	// The current account the user is operating in.
	CurrentAccount *TenancyCurrentAccount `json:"current_account"`
	// The sandbox accounts available to the user.
	Sandboxes []TenancySandboxAccount `json:"sandboxes" validate:"required"`
	// The owner account for the user's tenancy.
	OwnerAccount *TenancyOwnerAccount `json:"owner_account"`
	// Other accounts the user has access to.
	OtherAccounts []TenancyOtherAccount `json:"other_accounts" validate:"required"`
}

// TenancyCurrentAccount represents the account the user is currently operating in.
type TenancyCurrentAccount struct {
	// The unique identifier for the account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The display name of the account.
	Name string `json:"name" validate:"required"`
	// The type of the account.
	Type string `json:"type" validate:"required"`
	// The onboarding status of the account.
	OnboardingStatus string `json:"onboarding_status" validate:"required"`
	// The plan the account is on.
	Plan string `json:"plan" validate:"required"`
	// The account's unique slug.
	Slug *string `json:"slug"`
	// The user's role in this account.
	Role *Role `json:"role"`
}

// TenancySandboxAccount represents a sandbox account available to the user.
type TenancySandboxAccount struct {
	// The unique identifier for the sandbox account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The display name of the sandbox account.
	Name string `json:"name" validate:"required"`
}

// TenancyOwnerAccount represents the owner account for the user's tenancy.
type TenancyOwnerAccount struct {
	// The unique identifier for the owner account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The display name of the owner account.
	Name string `json:"name" validate:"required"`
}

// TenancyOtherAccount represents another account the user has access to.
type TenancyOtherAccount struct {
	// The unique identifier for the account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The display name of the account.
	Name string `json:"name" validate:"required"`
	// The type of the account.
	Type string `json:"type" validate:"required"`
}

func (*Tenancy) SchemaExample() any {
	return map[string]any{
		"object":          "tenancy",
		"current_account": nil,
		"sandboxes":       []any{},
		"owner_account":   nil,
		"other_accounts":  []any{},
	}
}

// CustomerAccountSummary represents a minimal customer account summary.
type CustomerAccountSummary struct {
	// The unique identifier for the account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The display name of the account.
	Name string `json:"name" validate:"required"`
}
