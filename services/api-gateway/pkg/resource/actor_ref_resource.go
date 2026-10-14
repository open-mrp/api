package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
)

// Who caused a record: a user, API key, agent, device, or OpenMRP itself.
//
// What happened is described by the record's own `type`; `actor` only says who did it.
type ActorRef struct {
	// What kind of actor this is.
	//
	// - `user`: a person signed in to OpenMRP.
	// - `api_key`: a program calling the API with an API key.
	// - `agent`: an OpenMRP agent acting for the account.
	// - `device`: an enrolled device, such as a scanning station.
	// - `system`: OpenMRP itself, such as a scheduled job. It has no `id`.
	Type constants.ActorType `json:"type" validate:"required"`
	// ID of the user, API key, agent or device. `null` for `system`.
	ID *string `json:"id"`
	// Display name: the person's or agent's name, or the API key's or device's label. `null` when the actor no longer exists.
	Name *string `json:"name"`
	// The client an API key belongs to, such as `Claude Code`. `null` for other actor types, and for keys created without one.
	Client *string `json:"client"`
}

var SampleActorRef = &ActorRef{
	Type:   constants.ActorTypeAPIKey,
	ID:     new(SampleAPIKeyID),
	Name:   new("Order sync"),
	Client: new("Claude Code"),
}

func (*ActorRef) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleActorRef)
}
