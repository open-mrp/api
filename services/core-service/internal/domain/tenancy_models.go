package domain

import "time"

// TenancyAccount represents an enriched account row from the tenancy query,
// containing all fields needed for tenancy resolution.
type TenancyAccount struct {
	AccountID                string
	AccountName              string
	AccountTypeCode          string
	OnboardingStatusCode     string
	PlanCode                 string
	RoleID                   *string
	RoleName                 *string
	RoleType                 *string
	RoleCreatedAt            *time.Time
	RoleUpdatedAt            *time.Time
	AccountUserID            string
	AccountUserStatusCode    string
	LastUsedAt               *time.Time
	OwnerAccountID           *string
	InternalStripeCustomerID *string
	Plan                     *TenancyAccountPlanSummary
}

// TenancyAccountPlanSummary is the inline plan data returned by the tenancy query,
// before limits/features have been joined in.
type TenancyAccountPlanSummary struct {
	TypeID        string
	Name          string
	PlanTypeCode  string
	Version       int32
	PricePerSeat  float64
	PricePerMonth *float64
	SeatMinimum   *int32
}

// Tenancy is the resolved multi-tenant context for a user.
type Tenancy struct {
	HasTenancy          bool
	CurrentAccount      *TenancyCurrentAccount
	Sandboxes           []TenancySandbox
	OwnerAccount        *TenancyOwnerAccount
	OtherAccounts       []TenancyOtherAccount
	PendingRegistration *TenancyPendingRegistration
}

// TenancyCurrentAccount represents the user's active account.
type TenancyCurrentAccount struct {
	ID                       string
	Name                     string
	Type                     string
	OnboardingStatus         string
	PlanCode                 string
	Slug                     *string
	Role                     *TenancyRole
	InternalStripeCustomerID *string
	AccountPlan              *TenancyAccountPlan
}

// TenancyAccountPlan is the fully-resolved plan for the current account,
// including its limits and features.
type TenancyAccountPlan struct {
	TypeID        string
	Name          string
	PlanTypeCode  string
	Version       int32
	PricePerSeat  float64
	PricePerMonth *float64
	SeatMinimum   *int32
	// Keys are limit codes, values are the limit int (nil = unlimited).
	Limits map[string]*int32
	// Keys are feature codes, values indicate whether the feature is enabled.
	Features map[string]bool
}

// TenancyRole represents the user's role on the current account.
type TenancyRole struct {
	ID          string
	Name        string
	RoleType    string
	Permissions []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TenancyPendingRegistration represents an in-progress registration session
// for the authenticated user.
type TenancyPendingRegistration struct {
	SessionID string
	PlanCode  string
	Step      string
	CreatedAt time.Time
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
