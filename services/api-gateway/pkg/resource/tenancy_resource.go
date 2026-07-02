package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// The authenticated user's tenancy context: which account they are currently acting in and every other account they can switch to.
type Tenancy struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=tenancy"`
	// Account the request is currently acting within.
	//
	// Absent when the user has no active account selected, such as partway through registration.
	CurrentAccount *TenancyCurrentAccount `json:"current_account"`
	// Sandbox accounts the user can switch into.
	Sandboxes *List[TenancySandboxAccount] `json:"sandboxes" validate:"required"`
	// The user's own production account, the account they signed up to create.
	OwnerAccount *TenancyOwnerAccount `json:"owner_account"`
	// Accounts the user has been granted access to beyond their current and owner accounts, such as other vendors' accounts.
	OtherAccounts *List[TenancyOtherAccount] `json:"other_accounts" validate:"required"`
	// In-progress registration session, populated only partway through signup before the account exists.
	PendingRegistration *TenancyPendingRegistration `json:"pending_registration"`
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
	//
	// - `company`: a standard production account.
	// - `sandbox`: an isolated testing account.
	Type string `json:"type" validate:"required"`
	// Onboarding status.
	OnboardingStatus string `json:"onboarding_status" validate:"required"`
	// Code of the account's subscription plan (for example `free`, `starter`, or `pro`).
	Plan string `json:"plan" validate:"required"`
	// The account's customer portal slug.
	Slug *string `json:"slug"`
	// The authenticated user's role in this account.
	Role *Role `json:"role"`
	// Internal Stripe customer ID for this account.
	InternalStripeCustomerID *string `json:"internal_stripe_customer_id"`
	// Full plan details for this account, including limits and features.
	AccountPlan *TenancyAccountPlan `json:"account_plan"`
	// ID of the authenticated user's membership record within this account.
	AccountUserID string `json:"account_user_id"`
}

// The resolved subscription plan for the current account, including its limits and features.
type TenancyAccountPlan struct {
	// Plan ID.
	TypeID string `json:"type_id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_plan"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// Plan type code.
	PlanTypeCode string `json:"plan_type_code" validate:"required"`
	// Plan version.
	Version int32 `json:"version"`
	// Price per seat per month.
	PricePerSeat float64 `json:"price_per_seat"`
	// Flat monthly price, if applicable.
	PricePerMonth *float64 `json:"price_per_month"`
	// Minimum seats required for this plan.
	SeatMinimum *int32 `json:"seat_minimum"`
	// Resource limits, keyed by limit code; a `null` value means unlimited.
	Limits map[string]*int32 `json:"limits"`
	// Feature availability for this plan, keyed by feature code.
	Features map[string]bool `json:"features"`
}

// An in-progress registration session, present only partway through signup before an account exists.
type TenancyPendingRegistration struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=tenancy_pending_registration"`
	// Registration session ID.
	SessionID string `json:"session_id" validate:"required"`
	// Plan code selected during registration.
	PlanCode string `json:"plan_code" validate:"required"`
	// Current step in the registration flow.
	Step string `json:"step" validate:"required"`
	// Session creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
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
	//
	// - `company`: a standard production account.
	// - `sandbox`: an isolated testing account.
	Type string `json:"type" validate:"required"`
}

// SampleTenancy describes a fully registered user operating in their owner account, so pending_registration is null — a pending registration only exists mid-signup, before the account does.
var SampleTenancy = &Tenancy{
	Object: constants.ObjectTypeTenancy,
	CurrentAccount: &TenancyCurrentAccount{
		ID:               SampleAccountID,
		Object:           constants.ObjectTypeAccount,
		Name:             SampleAccountName,
		Type:             string(constants.AccountTypeCodeStandard),
		OnboardingStatus: "active",
		Plan:             string(constants.PublicPlanCodeStarter),
		Role:             SampleRole,
		AccountUserID:    SampleAccountUserID,
		AccountPlan: &TenancyAccountPlan{
			TypeID:        SamplePlanTypeIDStarter,
			Object:        constants.ObjectTypeAccountPlan,
			Name:          "Starter",
			PlanTypeCode:  string(constants.PublicPlanCodeStarter),
			Version:       1,
			PricePerSeat:  19,
			PricePerMonth: new(19.0),
			SeatMinimum:   new(int32(1)),
			Limits: map[string]*int32{
				"sandboxes_maximum": new(int32(3)),
				"seats_maximum":     new(int32(5)),
				"invoices_maximum":  nil,
			},
			Features: map[string]bool{
				"customer_portal":      true,
				"sales_rep_dashboards": false,
			},
		},
	},
	Sandboxes: NewList([]TenancySandboxAccount{}, PageInfo{}),
	OwnerAccount: &TenancyOwnerAccount{
		ID:     SampleAccountID,
		Object: constants.ObjectTypeAccount,
		Name:   SampleAccountName,
	},
	OtherAccounts: NewList([]TenancyOtherAccount{}, PageInfo{}),
}

func (*Tenancy) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleTenancy)
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
