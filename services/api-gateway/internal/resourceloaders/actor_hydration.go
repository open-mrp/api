package resourceloaders

import (
	"context"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
)

// HydrateActorNames fills display name + handle (+ avatar for users) on the given actors by batch-resolving their backing ids: user actors via core-service, agent actors via agent-service.
// Group/system personas already carry their display name, so they are left untouched. Best-effort:
// on failure the actors keep whatever names they already had. Safe to pass nils and duplicates.
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
