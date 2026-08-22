package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
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
	//
	// Only administrators of a production account see its sandboxes; the list is empty for every other role, and while already acting inside a sandbox.
	Sandboxes *List[TenancySandboxAccount] `json:"sandboxes" validate:"required"`
	// The production account that the current account belongs to.
	//
	// This is the current account itself when acting in a production account, and its parent account when acting inside a sandbox.
	OwnerAccount *TenancyOwnerAccount `json:"owner_account"`
	// Accounts the user has been granted access to beyond their current and owner accounts, such as other vendors' accounts.
	//
	// Only accounts that have finished onboarding appear here, and the current account's own sandboxes are listed under `sandboxes` instead of being duplicated into this list.
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
	// How far the account has progressed through onboarding.
	//
	// The account is fully set up and usable once this is `active`.
	OnboardingStatus string `json:"onboarding_status" validate:"required"`
	// Code of the account's subscription plan (for example `free`, `starter`, or `pro`).
	//
	// The same code appears as `account_plan.plan_type_code`, alongside the plan's resolved limits and features.
	Plan string `json:"plan" validate:"required"`
	// The slug this account's customer portal is addressed by.
	//
	// Absent until the account enables its customer portal.
	Slug *string `json:"slug"`
	// The authenticated user's role in this account.
	Role *Role `json:"role"`
	// The Stripe customer that OpenMRP bills this account's own subscription and usage against.
	//
	// This is not the account's own Stripe customer for charging their customers.
	InternalStripeCustomerID *string `json:"internal_stripe_customer_id"`
	// Full plan details for this account, including limits and features.
	AccountPlan *TenancyAccountPlan `json:"account_plan"`
	// ID of the authenticated user's membership record within this account.
	AccountUserID string `json:"account_user_id"`
}

// The resolved subscription plan for the current account, including its limits and features.
type TenancyAccountPlan struct {
	// Identifier of the plan definition the account is subscribed to.
	TypeID string `json:"type_id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_plan"`
	// Display name of the plan, as shown in billing.
	Name string `json:"name" validate:"required"`
	// Stable code for the plan tier (for example `free`, `starter`, or `pro`).
	PlanTypeCode string `json:"plan_type_code" validate:"required"`
	// Revision of the plan definition the account is on.
	//
	// Plans are versioned so existing subscribers keep the pricing, limits, and features they signed up under when a newer version of the same plan is published.
	Version int32 `json:"version"`
	// Price per seat per month in dollars.
	PricePerSeat float64 `json:"price_per_seat"`
	// Per-seat price override in dollars used in place of `price_per_seat` when set.
	//
	// The monthly bill multiplies this by the number of seats (at least `seat_minimum`). `null` or `0` falls back to `price_per_seat`.
	PricePerMonth *float64 `json:"price_per_month"`
	// Fewest seats the account is billed for, regardless of how many users it actually has.
	SeatMinimum *int32 `json:"seat_minimum"`
	// Ceilings this plan imposes, keyed by limit code (for example `seats_maximum`).
	//
	// A `null` value means that resource is unlimited on this plan.
	Limits map[string]*int32 `json:"limits"`
	// Which capabilities this plan unlocks, keyed by feature code (for example `customer_portal`).
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
	// How far the signup has progressed, so the flow can be resumed where the user left off.
	//
	// Steps run `verification`, `user_details`, `account_details`, `review`, `payment`, then `completed`.
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

// The production account that the current account belongs to.
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

// A customer account under a vendor that the authenticated user is able to act on behalf of in that vendor's customer portal.
type CustomerAccountSummary struct {
	// Account ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// Display name.
	Name string `json:"name" validate:"required"`
}
