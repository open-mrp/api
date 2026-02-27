package apiresource

import apiexample "github.com/augno/api/services/api-gateway/pkg/example"

// AccountAffiliationRole represents a minimal role reference within an account affiliation.
type AccountAffiliationRole struct {
	// The unique identifier for the role.
	ID string `json:"id" validate:"required"`
	// The display name of the role.
	Name string `json:"name" validate:"required"`
}

// AccountAffiliation represents an account the user is a member of.
type AccountAffiliation struct {
	// The unique identifier of the affiliated account.
	ID string `json:"id" validate:"required"`
	// The display name of the affiliated account.
	Name string `json:"name" validate:"required"`
	// The user's role within this account.
	Role AccountAffiliationRole `json:"role" validate:"required"`
}

var SampleAccountAffiliation = &AccountAffiliation{
	ID:   SampleAccountID,
	Name: SampleAccountName,
	Role: AccountAffiliationRole{
		ID:   SampleRoleID,
		Name: SampleRoleName,
	},
}

func (*AccountAffiliation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountAffiliation)
}
