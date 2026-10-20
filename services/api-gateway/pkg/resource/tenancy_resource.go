package apiresource

import (
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
	Type constants.AccountTypeCode `json:"type" validate:"required"`
	// How far the account has progressed through onboarding.
	//
	// The account is fully set up and usable once this is `active`.
	OnboardingStatus constants.OnboardingStatus `json:"onboarding_status" validate:"required"`
	// The slug this account's customer portal is addressed by.
	//
	// Absent until the account enables its customer portal.
	Slug *string `json:"slug"`
	// The authenticated user's role in this account.
	Role *Role `json:"role"`
	// ID of the authenticated user's membership record within this account.
	AccountUserID string `json:"account_user_id"`
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
	Type constants.AccountTypeCode `json:"type" validate:"required"`
}

var SampleTenancy = &Tenancy{
	Object: constants.ObjectTypeTenancy,
	CurrentAccount: &TenancyCurrentAccount{
		ID:               SampleAccountID,
		Object:           constants.ObjectTypeAccount,
		Name:             SampleAccountName,
		Type:             constants.AccountTypeCodeStandard,
		OnboardingStatus: constants.OnboardingStatusActive,
		Role:             SampleRole,
		AccountUserID:    SampleAccountUserID,
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
