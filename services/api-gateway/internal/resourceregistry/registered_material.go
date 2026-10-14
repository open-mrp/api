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
		ObjectType: constants.ObjectTypeMaterial,
		Load:       resourceloaders.LoadMaterials,
		Subs: []include.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromMaterial,
				Populate:    populateItemOnMaterial,
			},
		},
	})
}

func extractItemIDFromMaterial(ctx context.Context, parent any) []string {
	m := parent.(*apiresource.Material)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMaterial, m.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnMaterial(ctx context.Context, parent any, loaded map[string]any) {
	m := parent.(*apiresource.Material)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMaterial, m.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		m.Item = v.(*apiresource.Item)
	}
}
