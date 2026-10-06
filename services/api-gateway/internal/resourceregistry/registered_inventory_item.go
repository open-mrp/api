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
		// `quantity` is a ComputedQuantity whose unit the presenter resolves in full, so `quantity.unit`
		// is not offered: the stored-quantity resolver it would point at panics on a computed one.
		Subs: []resourcekit.SubField{
			{
				Key:         "product_line",
				Target:      constants.ObjectTypeProductLine,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractProductLineIDFromInventoryItem,
				Populate:    populateProductLineOnInventoryItem,
			},
		},
	})
}

// inventoryItemProductLineID reads the line the presenter stashed, keyed by the item the row reports on.
func inventoryItemProductLineID(ctx context.Context, row *apiresource.InventoryItem) string {
	if row.Item == nil {
		return ""
	}
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeInventoryItem, row.Item.ID, "product_line_id")
	return id
}

func extractProductLineIDFromInventoryItem(ctx context.Context, parent any) []string {
	row, ok := parent.(*apiresource.InventoryItem)
	if !ok {
		return nil
	}
	if id := inventoryItemProductLineID(ctx, row); id != "" {
		return []string{id}
	}
	return nil
}

func populateProductLineOnInventoryItem(ctx context.Context, parent any, loaded map[string]any) {
	row, ok := parent.(*apiresource.InventoryItem)
	if !ok {
		return
	}
	if v, ok := loaded[inventoryItemProductLineID(ctx, row)]; ok {
		if line, ok := v.(*apiresource.ProductLine); ok {
			row.ProductLine = line
		}
	}
}
