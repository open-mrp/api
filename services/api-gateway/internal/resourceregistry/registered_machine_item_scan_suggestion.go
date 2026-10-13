package resourceregistry

import (
	"context"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeMachineItemScanSuggestion,
		// A suggestion is computed by its endpoint, never fetched by id.
		Load: stubLoader,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromMachineItemScanSuggestion,
				Populate:    populateItemOnMachineItemScanSuggestion,
			},
		},
	})
}

func extractItemIDFromMachineItemScanSuggestion(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.MachineItemScanSuggestion)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachineItemScanSuggestion, s.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnMachineItemScanSuggestion(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.MachineItemScanSuggestion)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachineItemScanSuggestion, s.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.Item = v.(*apiresource.Item)
	}
}
