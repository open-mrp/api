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
		ObjectType: constants.ObjectTypeQuantity,
		Load:       resourceloaders.LoadQuantities,
		Subs: []resourcekit.SubField{
			{Key: "unit", Populate: populateUnitOnQuantity},
		},
	})
}

func populateUnitOnQuantity(ctx context.Context, parent any, _ map[string]any) {
	q := parent.(*apiresource.Quantity)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeQuantity, q.ID, "unit")
	if !ok {
		return
	}
	q.Unit = v.(*apiresource.Unit)
}
