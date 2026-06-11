package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Reference to an actor (user, API key, or agent).
type Actor struct {
	// Actor ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=actor"`
	// Actor type.
	//
	// - `user`: a human user account.
	// - `api_key`: a programmatic caller authenticating with an API key.
	// - `agent`: an automated agent acting on the account's behalf.
	Type constants.ActorType `json:"type" validate:"required"`
	// The actor's display name.
	Name *string `json:"name"`
	// Human-readable handle identifying the actor.
	//
	// - For `user` actors: the user's email address.
	// - For `api_key` actors: the redacted key value.
	//
	// Agent actors carry no handle.
	Handle *string `json:"handle"`
	// Assigned role.
	Role *Role `json:"role" expandable:"true"`
}

// NewActor constructs an Actor reference with the canonical "actor" object type and the supplied actor subtype (user, api_key, agent).
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
