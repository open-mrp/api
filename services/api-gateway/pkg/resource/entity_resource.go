package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Entity is a polymorphic reference to any resource in the system.
type Entity struct {
	// Unique identifier for the entity.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=entity"`
	// The resource kind that this entity references (e.g. "user", "customer", "sales_order").
	Type constants.ObjectType `json:"type" validate:"required"`
}

// NewEntity constructs an Entity reference with the canonical "entity" object type
// and the supplied resource kind.
func NewEntity(id string, entityType constants.ObjectType) *Entity {
	return &Entity{
		ID:     id,
		Object: constants.ObjectTypeEntity,
		Type:   entityType,
	}
}

var SampleUserEntity = &Entity{
	ID:     SampleUserID,
	Object: constants.ObjectTypeEntity,
	Type:   constants.ObjectTypeUser,
}

var SampleCustomerEntity = &Entity{
	ID:     SampleCustomerID,
	Object: constants.ObjectTypeEntity,
	Type:   constants.ObjectTypeAccount,
}

func (*Entity) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserEntity)
}
