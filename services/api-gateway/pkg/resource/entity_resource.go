package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// Entity is a polymorphic reference to any resource in the system.
type Entity struct {
	// Unique identifier for the entity.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=entity"`
	// The resource kind that this entity references, as an object-type value (e.g. `user`, `account`).
	//
	// Unlike `object` — which is always `entity` — this names the underlying resource the `id` points to.
	Type constants.ObjectType `json:"type" validate:"required"`
	// Human-readable display name for the entity (e.g. a user's full name, a sales order number).
	Name *string `json:"name"`
	// Secondary human-readable identifier (e.g. email address, username, redacted API key value).
	Handle *string `json:"handle"`
}

// NewEntity constructs an Entity reference with the canonical "entity" object type and the supplied resource kind.
func NewEntity(id string, entityType constants.ObjectType, name, handle *string) *Entity {
	return &Entity{
		ID:     id,
		Object: constants.ObjectTypeEntity,
		Type:   entityType,
		Name:   name,
		Handle: handle,
	}
}

var SampleUserEntity = &Entity{
	ID:     SampleUserID,
	Object: constants.ObjectTypeEntity,
	Type:   constants.ObjectTypeUser,
	Name:   new(SampleUserName),
	Handle: new(SampleUserEmail),
}

var SampleCustomerEntity = &Entity{
	ID:     SampleCustomerID,
	Object: constants.ObjectTypeEntity,
	Type:   constants.ObjectTypeCustomer,
	Name:   new(SampleCustomerName),
	Handle: new(SampleCustomerNumber),
}

func (*Entity) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserEntity)
}
