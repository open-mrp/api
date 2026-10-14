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

// ResolveActorRefs fills each actor's display name in one batch per actor type: a person's name, an agent's name, an API key's or device's label. An actor that no longer exists keeps a null name. It is best-effort, so a failed lookup leaves names null rather than failing the response. Nils and duplicates are fine.
//
// `client` is filled only when the RPC already carried it: API keys gain a client with forge.1's agent work (G7).
func ResolveActorRefs(ctx context.Context, refs []*apiresource.ActorRef) {
	ids := map[string]map[string]struct{}{}
	add := func(kind, v string) {
		if ids[kind] == nil {
			ids[kind] = map[string]struct{}{}
		}
		ids[kind][v] = struct{}{}
	}
	for _, r := range refs {
		if r == nil || r.ID == nil || *r.ID == "" || r.Name != nil {
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
			if key, ok := v.(*apiresource.APIKey); ok && key.Name != "" {
				names[k] = key.Name
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
		if r == nil || r.ID == nil || r.Name != nil {
			continue
		}
		if n, ok := names[*r.ID]; ok {
			r.Name = new(n)
		}
	}
}
