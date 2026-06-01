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
		ObjectType: constants.ObjectTypeProduct,
		Load:       resourceloaders.LoadProducts,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromProduct,
				Populate:    populateItemOnProduct,
			},
			{
				Key:         "product_line",
				Target:      constants.ObjectTypeProductLine,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractProductLineIDFromProduct,
				Populate:    populateProductLineOnProduct,
			},
		},
	})
}

func extractItemIDFromProduct(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.Product)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProduct, p.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnProduct(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.Product)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProduct, p.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.Item = v.(*apiresource.Item)
	}
}

func extractProductLineIDFromProduct(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.Product)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProduct, p.ID, "product_line_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateProductLineOnProduct(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.Product)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProduct, p.ID, "product_line_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.ProductLine = v.(*apiresource.ProductLine)
	}
}
