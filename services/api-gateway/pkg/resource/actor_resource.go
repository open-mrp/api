package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Actor is a reference to an actor (user, API key, or agent).
type Actor struct {
	// The unique identifier for the actor.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=actor"`
	// The type of actor.
	Type constants.ActorType `json:"type" validate:"required"`
	// The display name of the actor.
	Name *string `json:"name"`
	// Human-readable handle (email for users, redacted value for API keys, slug for agents).
	Handle *string `json:"handle"`
	// The role assigned to the actor.
	Role *Role `json:"role" expandable:"true"`
}

// NewActor constructs an Actor reference with the canonical "actor" object type
// and the supplied actor subtype (user, api_key, agent).
func NewActor(id string, actorType constants.ActorType, name, handle *string) *Actor {
	return &Actor{
		ID:     id,
		Object: constants.ObjectTypeActor,
		Type:   actorType,
		Name:   name,
		Handle: handle,
	}
}

var SampleActor = &Actor{
	ID:     SampleUserID,
	Object: constants.ObjectTypeActor,
	Type:   constants.ActorTypeUser,
	Name:   new(SampleUserName),
	Handle: new(SampleUserEmail),
	Role:   SampleRole,
}

func (*Actor) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleActor)
}
