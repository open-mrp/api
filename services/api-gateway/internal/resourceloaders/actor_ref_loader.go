package resourceloaders

import (
	"context"
	"strings"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// ActorRefOf is the response shape of a stored actor. Name and client are left as the actor carries them; ResolveActorRefs fills them for a whole response at once.
func ActorRefOf(a actor.Actor) *apiresource.ActorRef {
	ref := &apiresource.ActorRef{Type: a.Type, Name: a.Name, Client: a.Client}
	if a.ID != "" {
		ref.ID = new(a.ID)
	}
	return ref
}

// ActorRefFromProto is the response shape of an actor an RPC carried, or nil when it carried none.
func ActorRefFromProto(p *pb.Actor) *apiresource.ActorRef {
	if p == nil || p.Type == "" {
		return nil
	}
	ref := &apiresource.ActorRef{Type: constants.ActorType(p.Type), Name: p.Name, Client: p.Client}
	if p.Id != nil && *p.Id != "" {
		ref.ID = new(*p.Id)
	}
	return ref
}

// ActorRefFromRecorded is the response shape of an actor a log recorded as a type and ID, or nil when the type isn't
// an actor type. A system actor's recorded ID names its process (`system:stripe`); the response has no ID for it and
// is named after the process. A recognized ID is enough on its own: it marks a system actor whatever type sits beside it.
func ActorRefFromRecorded(actorType constants.ActorType, actorID string, name *string) *apiresource.ActorRef {
	if process, ok := constants.ParseSystemActorID(actorID); ok || actorType == constants.ActorTypeSystem {
		display := constants.SystemActor("").DisplayName()
		if ok {
			display = process.DisplayName()
		}
		return &apiresource.ActorRef{Type: constants.ActorTypeSystem, Name: new(display)}
	}
	if !actorType.IsValid() || actorID == "" {
		return nil
	}
	ref := &apiresource.ActorRef{Type: actorType, ID: new(actorID)}
	if name != nil && *name != "" {
		ref.Name = new(*name)
	}
	return ref
}

// ActorRefFromBareID is the response shape of an actor stored as a bare identity-actor ID, with its type recovered
// from the ID's prefix (see ActorRefFromID), or nil when the prefix names no actor type.
func ActorRefFromBareID(actorID string) *apiresource.ActorRef {
	if a := ActorRefFromID(actorID); a != nil {
		return ActorRefFromRecorded(a.Type, a.ID, nil)
	}
	if _, ok := constants.ParseSystemActorID(actorID); ok {
		return ActorRefFromRecorded(constants.ActorTypeSystem, actorID, nil)
	}
	return nil
}

// ResolveActorRefs fills each actor's display name in one batch per actor type: a person's name, an agent's name, an API key's or device's label. An actor that no longer exists keeps a null name. It is best-effort, so a failed lookup leaves names null rather than failing the response. Nils and duplicates are fine.
//
// An API key's `client` is filled from the key too, when the RPC didn't carry it.
func ResolveActorRefs(ctx context.Context, refs []*apiresource.ActorRef) {
	ids := map[string]map[string]struct{}{}
	add := func(kind, v string) {
		if ids[kind] == nil {
			ids[kind] = map[string]struct{}{}
		}
		ids[kind][v] = struct{}{}
	}
	for _, r := range refs {
		if r == nil || r.ID == nil || *r.ID == "" {
			continue
		}
		if r.Name != nil && (r.Type != constants.ActorTypeAPIKey || r.Client != nil) {
			continue
		}
		switch r.Type {
		case constants.ActorTypeUser:
			if strings.HasPrefix(*r.ID, string(id.AccountUserIDPrefix)+"_") {
				add("account_user", *r.ID)
			} else {
				add("user", *r.ID)
			}
		case constants.ActorTypeAPIKey, constants.ActorTypeAgent, constants.ActorTypeDevice:
			add(string(r.Type), *r.ID)
		}
	}
	if len(ids) == 0 {
		return
	}

	names := map[string]string{}
	clients := map[string]string{}
	if loaded, apiErr := LoadUsers(ctx, setToSlice(ids["user"])); apiErr == nil {
		for k, v := range loaded {
			if u, ok := v.(*apiresource.User); ok && u.Name != nil {
				names[k] = *u.Name
			}
		}
	}
	if loaded, apiErr := LoadAccountUserNames(ctx, setToSlice(ids["account_user"])); apiErr == nil {
		for k, v := range loaded {
			if v.Name != nil {
				names[k] = *v.Name
			}
		}
	}
	if loaded, apiErr := LoadAPIKeys(ctx, setToSlice(ids[string(constants.ActorTypeAPIKey)])); apiErr == nil {
		for k, v := range loaded {
			key, ok := v.(*apiresource.APIKey)
			if !ok {
				continue
			}
			if key.Name != "" {
				names[k] = key.Name
			}
			if key.Client != nil {
				clients[k] = key.Client.Name
			}
		}
	}
	if loaded, apiErr := LoadAgentDefinitionNames(ctx, setToSlice(ids[string(constants.ActorTypeAgent)])); apiErr == nil {
		for k, v := range loaded {
			if v.Name != "" {
				names[k] = v.Name
			}
		}
	}
	// Devices are today's scanning stations until the device resource replaces them.
	if loaded, apiErr := LoadScanningStations(ctx, setToSlice(ids[string(constants.ActorTypeDevice)])); apiErr == nil {
		for k, v := range loaded {
			if s, ok := v.(*apiresource.ScanningStation); ok && s.Name != "" {
				names[k] = s.Name
			}
		}
	}

	for _, r := range refs {
		if r == nil || r.ID == nil {
			continue
		}
		if n, ok := names[*r.ID]; ok && r.Name == nil {
			r.Name = new(n)
		}
		if c, ok := clients[*r.ID]; ok && r.Type == constants.ActorTypeAPIKey && r.Client == nil {
			r.Client = new(c)
		}
	}
}
