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
		ObjectType: constants.ObjectTypeQuantity,
		Load:       resourceloaders.LoadQuantities,
		Subs: []include.SubField{
			{
				Key:         "unit",
				Target:      constants.ObjectTypeUnit,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractUnitIDFromQuantity,
				Populate:    populateUnitOnQuantity,
			},
		},
	})
}

func extractUnitIDFromQuantity(ctx context.Context, parent any) []string {
	q := parent.(*apiresource.Quantity)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeQuantity, q.ID, "unit_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateUnitOnQuantity(ctx context.Context, parent any, loaded map[string]any) {
	q := parent.(*apiresource.Quantity)
	meta := include.GetLoadMeta(ctx)
	// Loader path: a "unit_id" was stashed (proto carried only the id) — use the real Unit fetched by LoadUnits.
	if id, _ := meta.GetString(constants.ObjectTypeQuantity, q.ID, "unit_id"); id != "" {
		if v, ok := loaded[id]; ok {
			q.Unit = v.(*apiresource.Unit)
		}
		return
	}
	// Inline path: a fully-resolved Unit was stashed because the parent proto already carried complete, real unit detail (no extra fetch needed).
	if v, ok := meta.Get(constants.ObjectTypeQuantity, q.ID, "unit"); ok {
		q.Unit = v.(*apiresource.Unit)
	}
}
