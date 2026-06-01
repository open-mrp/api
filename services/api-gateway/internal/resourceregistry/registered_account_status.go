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
		ObjectType: constants.ObjectTypeAccountStatus,
		Load:       resourceloaders.LoadAccountStatuses,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnAccountStatus},
		},
	})
}

func populateOwnerOnAccountStatus(_ context.Context, parent any, _ map[string]any) {
	parent.(*apiresource.AccountStatus).Owner = &apiresource.Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeSystem,
	}
}
