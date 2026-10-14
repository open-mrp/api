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
		ObjectType: constants.ObjectTypeConsumption,
		Load:       resourceloaders.LoadConsumptions,
		Subs: []include.SubField{
			{Key: "consumed_item", Target: constants.ObjectTypeItem, Cardinality: include.CardinalityOnePtr, ExtractIDs: extractConsumedItemIDFromConsumption, Populate: populateConsumedItemOnConsumption},
		},
	})
}

func extractConsumedItemIDFromConsumption(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.Consumption)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeConsumption, c.ID, "consumed_item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateConsumedItemOnConsumption(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.Consumption)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeConsumption, c.ID, "consumed_item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		c.ConsumedItem = v.(*apiresource.Item)
	}
}
