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
		ObjectType: constants.ObjectTypeConsumption,
		Load:       resourceloaders.LoadConsumptions,
		Subs: []resourcekit.SubField{
			{Key: "consumed_item", Populate: populateConsumedItemOnConsumption},
		},
	})
}

func populateConsumedItemOnConsumption(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Consumption)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeConsumption, c.ID, "consumed_item")
	if !ok {
		return
	}
	c.ConsumedItem = v.(*apiresource.Item)
}
