package apiresource

import (
	"github.com/augno/api/shared/constants"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
)

// AccountAffiliationRole represents a minimal role reference within an account affiliation.
type AccountAffiliationRole struct {
	// The unique identifier for the role.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role"`
	// The display name of the role.
	Name string `json:"name" validate:"required"`
}

// AccountAffiliation represents an account the user is a member of.
type AccountAffiliation struct {
	// The unique identifier of the affiliated account.
	ID string `json:"id" validate:"required"`
	// String representing the object's type.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_affiliation"`
	// The display name of the affiliated account.
	Name string `json:"name" validate:"required"`
	// The user's role within this account.
	Role AccountAffiliationRole `json:"role" validate:"required"`
}

var SampleAccountAffiliation = &AccountAffiliation{
	ID:     SampleAccountID,
	Object: constants.ObjectTypeAccountAffiliation,
	Name:   SampleAccountName,
	Role: AccountAffiliationRole{
		ID:     SampleRoleID,
		Object: constants.ObjectTypeRole,
		Name:   SampleRoleName,
	},
}

func (*AccountAffiliation) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountAffiliation)
}
