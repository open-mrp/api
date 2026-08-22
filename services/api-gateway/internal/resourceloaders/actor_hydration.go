package resourceloaders

import (
	"context"
	"strings"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
)

// legacyAPIKeyIDPrefix is the api_key type_id prefix used before GenID adopted the composable "apke" prefix. Rows written under the legacy scheme (seeds and data migrated from the dashboard) still carry it, so actor recognition must accept both.
const legacyAPIKeyIDPrefix = "apky_"

// ActorRefFromID builds a minimal Actor reference from a bare identity-actor id by recognizing its prefix: us_/acus_ → user, apke_/apky_ → api_key, agdf_ → agent. Columns like machine downtime's reported_by_id store identity.Actor.ID with no actor type alongside, so the type must be recovered from the id itself. User actors can carry either a user id (us_, written by user-token identities) or an account_user id (acus_, written by legacy dashboard rows). Returns nil for an empty or unrecognized id rather than fabricating an actor of unknown type.
func ActorRefFromID(actorID string) *apiresource.Actor {
	var actorType constants.ActorType
	switch {
	case strings.HasPrefix(actorID, string(id.UserIDPrefix)+"_"),
		strings.HasPrefix(actorID, string(id.AccountUserIDPrefix)+"_"):
		actorType = constants.ActorTypeUser
	case strings.HasPrefix(actorID, string(id.APIKeyIDPrefix)+"_"),
		strings.HasPrefix(actorID, legacyAPIKeyIDPrefix):
		actorType = constants.ActorTypeAPIKey
	case strings.HasPrefix(actorID, string(id.AgentDefinitionIDPrefix)+"_"):
		actorType = constants.ActorTypeAgent
	default:
		return nil
	}
	return apiresource.NewActor(actorID, actorType, nil, nil)
}

// HydrateIdentityActorNames fills display name + handle (+ avatar for users) on actors built from bare identity-actor ids (ActorRefFromID). User actors carry either a user id (us_, resolved via the user loader) or an account_user id (acus_, resolved via the account-user loader); api_key actors resolve via the api-key loader and agent actors via agent-service. Best-effort: on failure the actors keep whatever names they already had. Safe to pass nils and duplicates.
func HydrateIdentityActorNames(ctx context.Context, actors []*apiresource.Actor) {
	if len(actors) == 0 {
		return
	}
	accountUserIDs := map[string]struct{}{}
	idsByType := map[constants.ActorType]map[string]struct{}{}
	for _, a := range actors {
		if a == nil || a.ID == "" {
			continue
		}
		switch a.Type {
		case constants.ActorTypeUser:
			if strings.HasPrefix(a.ID, string(id.AccountUserIDPrefix)+"_") {
				accountUserIDs[a.ID] = struct{}{}
				continue
			}
			fallthrough
		case constants.ActorTypeAPIKey, constants.ActorTypeAgent:
			if idsByType[a.Type] == nil {
				idsByType[a.Type] = map[string]struct{}{}
			}
			idsByType[a.Type][a.ID] = struct{}{}
		}
	}

	users := map[string]any{}
	if ids := setToSlice(idsByType[constants.ActorTypeUser]); len(ids) > 0 {
		if loaded, apiErr := LoadUsers(ctx, ids); apiErr == nil {
			users = loaded
		}
	}
	var accountUserNames map[string]AccountUserName
	if ids := setToSlice(accountUserIDs); len(ids) > 0 {
		if names, apiErr := LoadAccountUserNames(ctx, ids); apiErr == nil {
			accountUserNames = names
		}
	}
	apiKeys := map[string]any{}
	if ids := setToSlice(idsByType[constants.ActorTypeAPIKey]); len(ids) > 0 {
		if loaded, apiErr := LoadAPIKeys(ctx, ids); apiErr == nil {
			apiKeys = loaded
		}
	}
	var agentNames map[string]AgentDefinitionName
	if ids := setToSlice(idsByType[constants.ActorTypeAgent]); len(ids) > 0 {
		if names, apiErr := LoadAgentDefinitionNames(ctx, ids); apiErr == nil {
			agentNames = names
		}
	}

	meta := resourcekit.GetLoadMeta(ctx)
	for _, a := range actors {
		if a == nil {
			continue
		}
		switch a.Type {
		case constants.ActorTypeUser:
			if u, ok := users[a.ID].(*apiresource.User); ok {
				if u.Name != nil {
					a.Name = u.Name
				}
				if u.Email != nil && a.Handle == nil {
					a.Handle = u.Email
				}
				if u.ImageUrl != nil {
					a.AvatarURL = u.ImageUrl
				}
			} else if n, ok := accountUserNames[a.ID]; ok {
				if n.Name != nil {
					a.Name = n.Name
				}
				if n.Email != nil && a.Handle == nil {
					a.Handle = n.Email
				}
				if n.ImageURL != nil {
					a.AvatarURL = n.ImageURL
				}
				if n.RoleID != nil {
					meta.Set(constants.ObjectTypeActor, a.ID, "role_id", *n.RoleID)
				}
			}
		case constants.ActorTypeAPIKey:
			if k, ok := apiKeys[a.ID].(*apiresource.APIKey); ok {
				if k.Name != "" && a.Name == nil {
					name := k.Name
					a.Name = &name
				}
				if k.RedactedValue != "" && a.Handle == nil {
					handle := k.RedactedValue
					a.Handle = &handle
				}
				if roleID, ok := meta.GetString(constants.ObjectTypeAPIKey, a.ID, "role_id"); ok {
					meta.Set(constants.ObjectTypeActor, a.ID, "role_id", roleID)
				}
			}
		case constants.ActorTypeAgent:
			if n, ok := agentNames[a.ID]; ok {
				if n.Name != "" && a.Name == nil {
					name := n.Name
					a.Name = &name
				}
				if n.Slug != "" && a.Handle == nil {
					slug := n.Slug
					a.Handle = &slug
				}
			}
		}
	}
}

func setToSlice(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	return out
}

// HydrateActorNames fills display name + handle (+ avatar for users) on the given actors by batch-resolving their backing ids: user actors via core-service, agent actors via agent-service. Group/system personas already carry their display name, so they are left untouched. Best-effort: on failure the actors keep whatever names they already had. Safe to pass nils and duplicates.
func HydrateActorNames(ctx context.Context, actors []*apiresource.Actor) {
	if len(actors) == 0 {
		return
	}
	userIDs := make(map[string]struct{})
	agentIDs := make(map[string]struct{})
	for _, a := range actors {
		if a == nil || a.ID == "" {
			continue
		}
		switch a.Type {
		case constants.ActorTypeUser:
			userIDs[a.ID] = struct{}{}
		case constants.ActorTypeAgent:
			agentIDs[a.ID] = struct{}{}
		}
	}

	var userNames map[string]AccountUserName
	if len(userIDs) > 0 {
		ids := make([]string, 0, len(userIDs))
		for id := range userIDs {
			ids = append(ids, id)
		}
		if names, apiErr := LoadAccountUserNames(ctx, ids); apiErr == nil {
			userNames = names
		}
	}

	var agentNames map[string]AgentDefinitionName
	if len(agentIDs) > 0 {
		ids := make([]string, 0, len(agentIDs))
		for id := range agentIDs {
			ids = append(ids, id)
		}
		if names, apiErr := LoadAgentDefinitionNames(ctx, ids); apiErr == nil {
			agentNames = names
		}
	}

	for _, a := range actors {
		if a == nil {
			continue
		}
		switch a.Type {
		case constants.ActorTypeUser:
			if n, ok := userNames[a.ID]; ok {
				if n.Name != nil {
					a.Name = n.Name
				}
				if n.Email != nil && a.Handle == nil {
					a.Handle = n.Email
				}
				if n.ImageURL != nil {
					a.AvatarURL = n.ImageURL
				}
			}
		case constants.ActorTypeAgent:
			if n, ok := agentNames[a.ID]; ok {
				if n.Name != "" && a.Name == nil {
					name := n.Name
					a.Name = &name
				}
				if n.Slug != "" && a.Handle == nil {
					slug := n.Slug
					a.Handle = &slug
				}
			}
		}
	}
}
