package resourceregistry

import (
	"context"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeItemLotDefault,
		// A lot default is never fetched by id — it is only ever the body of the endpoint that resolves it — but the registry requires a loader, so this one answers nothing. Only the unit sub-field does real work.
		Load: loadItemLotDefaults,
		Subs: []resourcekit.SubField{
			{
				Key:         "unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractUnitIDFromItemLotDefault,
				Populate:    populateUnitOnItemLotDefault,
			},
		},
	})
}

func loadItemLotDefaults(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

// The unit id lives in load meta rather than on the resource — a bare unit_id field would inline a foreign key the nested `unit` sub-resource already represents. The lot has no id of its own, so the meta is keyed by the item the lot was resolved for.
func itemLotDefaultUnitID(ctx context.Context, lot *apiresource.ItemLotDefault) (string, bool) {
	if lot.Item == nil {
		return "", false
	}
	return resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeItemLotDefault, lot.Item.ID, "unit_id")
}

func extractUnitIDFromItemLotDefault(ctx context.Context, parent any) []string {
	lot, ok := parent.(*apiresource.ItemLotDefault)
	if !ok {
		return nil
	}
	unitID, ok := itemLotDefaultUnitID(ctx, lot)
	if !ok || unitID == "" {
		return nil
	}
	return []string{unitID}
}

func populateUnitOnItemLotDefault(ctx context.Context, parent any, loaded map[string]any) {
	lot, ok := parent.(*apiresource.ItemLotDefault)
	if !ok {
		return
	}
	unitID, ok := itemLotDefaultUnitID(ctx, lot)
	if !ok || unitID == "" {
		return
	}
	if v, ok := loaded[unitID]; ok {
		if unit, ok := v.(*apiresource.Unit); ok {
			lot.Unit = unit
		}
	}
}
