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
		ObjectType: constants.ObjectTypePart,
		Load:       resourceloaders.LoadParts,
		Subs: []include.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromPart,
				Populate:    populateItemOnPart,
			},
		},
	})
}

func extractItemIDFromPart(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.Part)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypePart, p.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnPart(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.Part)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypePart, p.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.Item = v.(*apiresource.Item)
	}
}
