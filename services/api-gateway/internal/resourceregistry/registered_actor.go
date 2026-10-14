package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeActor,
		Load:       resourceloaders.LoadActors,
		Subs: []include.SubField{
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromActor,
				Populate:    populateRoleOnActor,
			},
		},
	})
}

func extractRoleIDFromActor(ctx context.Context, parent any) []string {
	actor := parent.(*apiresource.Actor)
	id, ok := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeActor, actor.ID, "role_id")
	if !ok || id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnActor(ctx context.Context, parent any, loaded map[string]any) {
	actor := parent.(*apiresource.Actor)
	id, ok := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeActor, actor.ID, "role_id")
	if !ok || id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		actor.Role = v.(*apiresource.Role)
	}
}
