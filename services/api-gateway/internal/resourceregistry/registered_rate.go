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
		ObjectType: constants.ObjectTypeRate,
		Load:       resourceloaders.LoadRates,
		Subs: []resourcekit.SubField{
			{Key: "numerator_unit", Populate: populateNumeratorUnitOnRate},
			{Key: "denominator_unit", Populate: populateDenominatorUnitOnRate},
		},
	})
}

func populateNumeratorUnitOnRate(ctx context.Context, parent any, _ map[string]any) {
	r := parent.(*apiresource.Rate)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRate, r.ID, "numerator_unit")
	if !ok {
		return
	}
	r.NumeratorUnit = v.(*apiresource.Unit)
}

func populateDenominatorUnitOnRate(ctx context.Context, parent any, _ map[string]any) {
	r := parent.(*apiresource.Rate)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRate, r.ID, "denominator_unit")
	if !ok {
		return
	}
	r.DenominatorUnit = v.(*apiresource.Unit)
}
