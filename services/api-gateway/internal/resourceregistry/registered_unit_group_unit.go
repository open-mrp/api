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
		ObjectType: constants.ObjectTypeUnitGroupUnit,
		Load:       resourceloaders.LoadUnitGroupUnits,
		Subs: []resourcekit.SubField{
			{
				Key:         "unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractUnitIDFromUnitGroupUnit,
				Populate:    populateUnitOnUnitGroupUnit,
			},
		},
	})
}

func extractUnitIDFromUnitGroupUnit(ctx context.Context, parent any) []string {
	ugu := parent.(*apiresource.UnitGroupUnit)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroupUnit, ugu.ID, "unit_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateUnitOnUnitGroupUnit(ctx context.Context, parent any, loaded map[string]any) {
	ugu := parent.(*apiresource.UnitGroupUnit)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroupUnit, ugu.ID, "unit_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ugu.Unit = v.(*apiresource.Unit)
	}
}
