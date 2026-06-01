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
		ObjectType: constants.ObjectTypeSalesOrderStatus,
		Load:       resourceloaders.LoadSalesOrderStatuses,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnSalesOrderStatus},
		},
	})
}

func populateOwnerOnSalesOrderStatus(_ context.Context, parent any, _ map[string]any) {
	parent.(*apiresource.SalesOrderStatus).Owner = &apiresource.Owner{
		Object: constants.ObjectTypeOwner,
		Type:   constants.OwnerTypeSystem,
	}
}
