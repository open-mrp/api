// Package actor is who caused a record: a user, API key, agent, device, or OpenMRP itself. What happened is the
// record's own type; the actor only says who did it.
//
// A table stores its actor in two columns, named with the role's prefix when a row has more than one actor
// (`actor_*`, `reviewed_by_*`):
//
//	`actor_type` varchar(32) NOT NULL,   -- constants.ActorType
//	`actor_id` varchar(191) DEFAULT NULL, -- NULL only for system
//
// Names and API-key clients are resolved when a record is read, never stored with it.
package actor

import (
	"database/sql"
	"fmt"

	authtypes "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// Actor is who caused a record. Build one with New or System.
type Actor struct {
	Type constants.ActorType
	// ID of the user, API key, agent or device. Empty for system.
	ID string
	// Name is the display name, set only once resolved for a response.
	Name *string
	// Client is the client an API key belongs to, set only once resolved for an api_key actor.
	Client *string
}

// New is an actor of type t with the given ID.
func New(t constants.ActorType, id string) Actor {
	return Actor{Type: t, ID: id}
}

// System is OpenMRP itself acting, such as a scheduled job or a migration.
func System() Actor {
	return Actor{Type: constants.ActorTypeSystem}
}

// IsSystem reports whether OpenMRP itself acted.
func (a Actor) IsSystem() bool {
	return a.Type == constants.ActorTypeSystem
}

// Validate reports an actor that can't be stored: an unknown type, a system actor with an ID, another actor
// without one, or a client on anything but an API key.
func (a Actor) Validate() error {
	switch a.Type {
	case constants.ActorTypeSystem:
		if a.ID != "" {
			return fmt.Errorf("actor: a system actor has no id, got %q", a.ID)
		}
	case constants.ActorTypeUser, constants.ActorTypeAPIKey, constants.ActorTypeAgent, constants.ActorTypeDevice:
		if a.ID == "" {
			return fmt.Errorf("actor: a %s actor needs an id", a.Type)
		}
	default:
		return fmt.Errorf("actor: unknown actor type %q", a.Type)
	}
	if a.Client != nil && a.Type != constants.ActorTypeAPIKey {
		return fmt.Errorf("actor: only an api_key actor has a client, got a %s actor", a.Type)
	}
	return nil
}

// Columns is the actor as its actor_type and actor_id columns store it.
func (a Actor) Columns() (actorType string, actorID sql.NullString) {
	return string(a.Type), sql.NullString{String: a.ID, Valid: a.ID != ""}
}

// FromColumns reads an actor from its actor_type and actor_id columns.
func FromColumns(actorType string, actorID sql.NullString) (Actor, error) {
	a := Actor{Type: constants.ActorType(actorType)}
	if actorID.Valid {
		a.ID = actorID.String
	}
	if err := a.Validate(); err != nil {
		return Actor{}, err
	}
	return a, nil
}

// FromIdentity is the actor behind a request or background write. A caller that isn't signed in has none.
func FromIdentity(identity *authtypes.Identity) (Actor, bool) {
	if identity == nil {
		return Actor{}, false
	}
	if identity.IsSystem() {
		return System(), true
	}
	if !identity.IsAuthenticated() || identity.Actor == nil || identity.Actor.ID == "" {
		return Actor{}, false
	}
	switch identity.Type {
	case authtypes.IdentityActorTypeUser:
		return New(constants.ActorTypeUser, identity.Actor.ID), true
	case authtypes.IdentityActorTypeAPIKey:
		return New(constants.ActorTypeAPIKey, identity.Actor.ID), true
	case authtypes.IdentityActorTypeAgent:
		return New(constants.ActorTypeAgent, identity.Actor.ID), true
	default:
		return Actor{}, false
	}
}

// ToProto is the actor as the RPC carries it.
func (a Actor) ToProto() *pb.Actor {
	p := &pb.Actor{Type: string(a.Type), Name: a.Name, Client: a.Client}
	if a.ID != "" {
		p.Id = &a.ID
	}
	return p
}

// FromProto is the RPC's actor, or nil when it was not sent.
func FromProto(p *pb.Actor) (*Actor, error) {
	if p == nil {
		return nil, nil
	}
	a := Actor{Type: constants.ActorType(p.Type), ID: p.GetId(), Name: p.Name, Client: p.Client}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	return &a, nil
}
