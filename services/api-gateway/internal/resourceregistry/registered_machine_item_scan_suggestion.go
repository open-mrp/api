package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeMachineItemScanSuggestion,
		// A suggestion is computed by its endpoint, never fetched by id.
		Load: stubLoader,
		Subs: []include.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractItemIDFromMachineItemScanSuggestion,
				Populate:    populateItemOnMachineItemScanSuggestion,
			},
		},
	})
}

func extractItemIDFromMachineItemScanSuggestion(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.MachineItemScanSuggestion)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachineItemScanSuggestion, s.ID, "item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateItemOnMachineItemScanSuggestion(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.MachineItemScanSuggestion)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeMachineItemScanSuggestion, s.ID, "item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.Item = v.(*apiresource.Item)
	}
}
