package domain

import "time"

// TenancyAccount represents an enriched account row from the tenancy query,
// containing all fields needed for tenancy resolution.
type TenancyAccount struct {
	AccountID             string
	AccountName           string
	AccountTypeCode       string
	OnboardingStatusCode  string
	PlanCode              string
	RoleID                *string
	RoleName              *string
	RoleTypeCode          *string
	AccountUserID         string
	AccountUserStatusCode string
	LastUsedAt            *time.Time
	OwnerAccountID        *string
}

// Tenancy is the resolved multi-tenant context for a user.
type Tenancy struct {
	HasTenancy     bool
	CurrentAccount *TenancyCurrentAccount
	Sandboxes      []TenancySandbox
	OwnerAccount   *TenancyOwnerAccount
	OtherAccounts  []TenancyOtherAccount
}

// TenancyCurrentAccount represents the user's active account.
type TenancyCurrentAccount struct {
	ID               string
	Name             string
	Type             string
	OnboardingStatus string
	PlanCode         string
	Slug             *string
	Role             *TenancyRole
}

// TenancyRole represents the user's role on the current account.
type TenancyRole struct {
	ID           string
	Name         string
	RoleTypeCode string
}

// TenancySandbox represents a sandbox account summary.
type TenancySandbox struct {
	ID   string
	Name string
}

// TenancyOwnerAccount represents the owner (production) account summary.
type TenancyOwnerAccount struct {
	ID   string
	Name string
}

// TenancyOtherAccount represents another accessible account.
type TenancyOtherAccount struct {
	ID   string
	Name string
	Type string
}

// CustomerAccountSummary represents a customer account summary.
type CustomerAccountSummary struct {
	ID   string
	Name string
}
