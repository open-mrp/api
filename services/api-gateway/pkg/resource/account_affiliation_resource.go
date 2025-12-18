package apiresource

import apiexample "github.com/augno/api/services/api-gateway/pkg/example"

// A minimal role for an account affiliation
type AccountAffiliationRole struct {
	// The ID of the role
	ID string `json:"id" validate:"required"`
	// The name of the role
	Name string `json:"name" validate:"required"`
}

// An affiliated account with the user
type AccountAffiliation struct {
	// The ID of the account affiliation
	ID string `json:"id" validate:"required"`
	// The name of the account affiliation
	Name string `json:"name" validate:"required"`
	// The role of the account affiliation
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
