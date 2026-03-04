package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

const SampleAccountID = "ac_01gf7a8200eaj8fke1xvw4h50x"
const SampleAccountName = "Acme Inc."

var SampleLightAccount = &LightAccount{
	ID:     SampleAccountID,
	Name:   SampleAccountName,
	Object: constants.ObjectTypeAccount,
}

// LightAccount represents a minimal account reference.
type LightAccount struct {
	// The unique identifier for the account.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account"`
	// The display name of the account.
	Name string `json:"name" validate:"required"`
}

func (*LightAccount) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleLightAccount)
}
