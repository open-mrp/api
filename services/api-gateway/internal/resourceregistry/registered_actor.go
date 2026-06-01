package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeActor,
		Load:       resourceloaders.LoadActors,
		Subs: []resourcekit.SubField{
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromActor,
				Populate:    populateRoleOnActor,
			},
		},
	})
}

func extractRoleIDFromActor(ctx context.Context, parent any) []string {
	actor := parent.(*apiresource.Actor)
	id, ok := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeActor, actor.ID, "role_id")
	if !ok || id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnActor(ctx context.Context, parent any, loaded map[string]any) {
	actor := parent.(*apiresource.Actor)
	id, ok := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeActor, actor.ID, "role_id")
	if !ok || id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		actor.Role = v.(*apiresource.Role)
	}
}
