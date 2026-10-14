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
		ObjectType: constants.ObjectTypeUnitGroupUnit,
		Load:       resourceloaders.LoadUnitGroupUnits,
		Subs: []include.SubField{
			{
				Key:         "unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractUnitIDFromUnitGroupUnit,
				Populate:    populateUnitOnUnitGroupUnit,
			},
		},
	})
}

func extractUnitIDFromUnitGroupUnit(ctx context.Context, parent any) []string {
	ugu := parent.(*apiresource.UnitGroupUnit)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroupUnit, ugu.ID, "unit_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateUnitOnUnitGroupUnit(ctx context.Context, parent any, loaded map[string]any) {
	ugu := parent.(*apiresource.UnitGroupUnit)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnitGroupUnit, ugu.ID, "unit_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ugu.Unit = v.(*apiresource.Unit)
	}
}
