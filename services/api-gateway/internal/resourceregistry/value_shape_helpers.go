package resourceregistry

import (
	"context"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
)

// valueShapeSub declares an expandable value-shape field (a Quantity, Rate, UnitPrice or Money) that the parent's
// presenter stashed in LoadMeta under its key. Its units load with the parent's other includes, so the field arrives
// with them in full.
func valueShapeSub(parentType constants.ObjectType, key string, idOf func(parent any) string, set func(parent, value any)) include.SubField {
	stashed := func(ctx context.Context, parent any) any {
		v, ok := include.GetLoadMeta(ctx).Get(parentType, idOf(parent), key)
		if !ok {
			return nil
		}
		return v
	}
	return include.SubField{
		Key:         key,
		Target:      constants.ObjectTypeUnit,
		Cardinality: include.CardinalityOnePtr,
		ExtractIDs: func(ctx context.Context, parent any) []string {
			return resourceloaders.UnitStubIDs(stashed(ctx, parent))
		},
		Populate: func(ctx context.Context, parent any, loaded map[string]any) {
			v := stashed(ctx, parent)
			if v == nil {
				return
			}
			resourceloaders.AttachUnits(v, loaded)
			set(parent, v)
		},
	}
}
