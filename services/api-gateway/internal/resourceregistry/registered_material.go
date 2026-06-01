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
		ObjectType: constants.ObjectTypeMaterial,
		Load:       resourceloaders.LoadMaterials,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromMaterial,
				Populate:    populateItemOnMaterial,
			},
		},
	})
}

func extractItemIDFromMaterial(ctx context.Context, parent any) []string {
	m := parent.(*apiresource.Material)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMaterial, m.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnMaterial(ctx context.Context, parent any, loaded map[string]any) {
	m := parent.(*apiresource.Material)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMaterial, m.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		m.Item = v.(*apiresource.Item)
	}
}
