package apiresource

import (
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
)

// CreatedBy describes who created a resource and their relationship to the account that owns it.
//
// It is resolved from the resource's create audit event.
type CreatedBy struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=created_by"`
	// The creator's relationship to the account that owns the resource.
	//
	// - `internal`: created by a user of the owning account.
	// - `customer`: created by a customer of the owning account.
	// - `system`: created automatically with no human actor (e.g. an EDI import).
	Relation constants.CreatedByRelation `json:"relation" validate:"required"`
	// The actor who created the resource.
	//
	// Absent when `relation` is `system`, since no human or programmatic actor performed the action.
	Actor *Actor `json:"actor"`
}

// NewCreatedBy builds a CreatedBy from a relation and optional actor.
func NewCreatedBy(relation constants.CreatedByRelation, actor *Actor) *CreatedBy {
	return &CreatedBy{
		Object:   constants.ObjectTypeCreatedBy,
		Relation: relation,
		Actor:    actor,
	}
}

// SystemCreatedBy returns a CreatedBy for a resource created with no human actor.
func SystemCreatedBy() *CreatedBy {
	return &CreatedBy{
		Object:   constants.ObjectTypeCreatedBy,
		Relation: constants.CreatedByRelationSystem,
	}
}

var SampleCreatedBy = &CreatedBy{
	Object:   constants.ObjectTypeCreatedBy,
	Relation: constants.CreatedByRelationInternal,
	Actor:    SampleActor,
}

func (*CreatedBy) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCreatedBy)
}
