package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeInventoryItem,
		Load:       resourceloaders.LoadInventoryItems,
		Subs: []resourcekit.SubField{
			{
				Key:         "quantity",
				Target:      constants.ObjectTypeQuantity,
				ExtractRefs: extractQuantityFromInventoryItem,
			},
		},
	})
}

func extractQuantityFromInventoryItem(_ context.Context, parent any) []any {
	item := parent.(*apiresource.InventoryItem)
	if item.Quantity == nil {
		return nil
	}
	return []any{item.Quantity}
}
