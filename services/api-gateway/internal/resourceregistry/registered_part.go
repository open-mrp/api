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
		ObjectType: constants.ObjectTypePart,
		Load:       resourceloaders.LoadParts,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromPart,
				Populate:    populateItemOnPart,
			},
		},
	})
}

func extractItemIDFromPart(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.Part)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypePart, p.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnPart(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.Part)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypePart, p.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.Item = v.(*apiresource.Item)
	}
}
