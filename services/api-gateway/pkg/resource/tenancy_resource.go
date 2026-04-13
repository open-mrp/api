package apiresource

import (
	"github.com/augno/api/shared/constants"
)

// Authenticated user's tenancy context.
type Tenancy struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=tenancy"`
	// Current account the user is operating in.
	CurrentAccount *TenancyCurrentAccount `json:"current_account"`
	// Sandbox accounts available to the user.
	Sandboxes []TenancySandboxAccount `json:"sandboxes" validate:"required"`
	// Owner account for the user's tenancy.
	OwnerAccount *TenancyOwnerAccount `json:"owner_account"`
	// Other accounts the user has access to.
	OtherAccounts []TenancyOtherAccount `json:"other_accounts" validate:"required"`
}

// Account the user is currently operating in.
type TenancyCurrentAccount struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Account type.
	Type string `json:"type" validate:"required"`
	// Onboarding status.
	OnboardingStatus string `json:"onboarding_status" validate:"required"`
	// Plan code.
	Plan string `json:"plan" validate:"required"`
	// Account slug.
	Slug *string `json:"slug"`
	// Role in this account.
	Role *Role `json:"role"`
}

// Sandbox account available to the user.
type TenancySandboxAccount struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

// Owner account for the user's tenancy.
type TenancyOwnerAccount struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name.
	Name string `json:"name" validate:"required"`
}

// Additional account the user has access to.
type TenancyOtherAccount struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Account type.
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

// Minimal customer account summary.
type CustomerAccountSummary struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name.
	Name string `json:"name" validate:"required"`
}
