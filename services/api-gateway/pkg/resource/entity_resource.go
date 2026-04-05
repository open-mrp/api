package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Entity represents an entity in the system.
type Entity struct {
	// The unique identifier for the entity.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required"`
}

var SampleUserEntity = &Entity{
	ID:     SampleUserID,
	Object: constants.ObjectTypeUser,
}

var SampleCustomerEntity = &Entity{
	ID:     SampleCustomerID,
	Object: constants.ObjectTypeAccount,
}

func (*Entity) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserEntity)
}
