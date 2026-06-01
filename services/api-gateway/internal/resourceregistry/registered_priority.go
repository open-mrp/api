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
		ObjectType: constants.ObjectTypePriority,
		Load:       resourceloaders.LoadPriorities,
		Subs: []resourcekit.SubField{
			// owner: priorities are always system-owned, so the projection
			// is a constant SystemOwner. No FK lookup needed.
			{Key: "owner", Populate: populateOwnerOnPriority},
		},
	})
}

func populateOwnerOnPriority(_ context.Context, parent any, _ map[string]any) {
	parent.(*apiresource.Priority).Owner = &apiresource.Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeSystem,
	}
}
