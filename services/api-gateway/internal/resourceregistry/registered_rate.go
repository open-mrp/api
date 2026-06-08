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
			{Key: "numerator_unit", Target: constants.ObjectTypeUnit, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractNumeratorUnitIDFromRate, Populate: populateNumeratorUnitOnRate},
			{Key: "denominator_unit", Target: constants.ObjectTypeUnit, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractDenominatorUnitIDFromRate, Populate: populateDenominatorUnitOnRate},
		},
	})
}

func extractNumeratorUnitIDFromRate(ctx context.Context, parent any) []string {
	r := parent.(*apiresource.Rate)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeRate, r.ID, "numerator_unit_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateNumeratorUnitOnRate(ctx context.Context, parent any, loaded map[string]any) {
	r := parent.(*apiresource.Rate)
	meta := resourcekit.GetLoadMeta(ctx)
	if id, _ := meta.GetString(constants.ObjectTypeRate, r.ID, "numerator_unit_id"); id != "" {
		if v, ok := loaded[id]; ok {
			r.NumeratorUnit = v.(*apiresource.Unit)
		}
		return
	}
	if v, ok := meta.Get(constants.ObjectTypeRate, r.ID, "numerator_unit"); ok {
		r.NumeratorUnit = v.(*apiresource.Unit)
	}
}

func extractDenominatorUnitIDFromRate(ctx context.Context, parent any) []string {
	r := parent.(*apiresource.Rate)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeRate, r.ID, "denominator_unit_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateDenominatorUnitOnRate(ctx context.Context, parent any, loaded map[string]any) {
	r := parent.(*apiresource.Rate)
	meta := resourcekit.GetLoadMeta(ctx)
	if id, _ := meta.GetString(constants.ObjectTypeRate, r.ID, "denominator_unit_id"); id != "" {
		if v, ok := loaded[id]; ok {
			r.DenominatorUnit = v.(*apiresource.Unit)
		}
		return
	}
	if v, ok := meta.Get(constants.ObjectTypeRate, r.ID, "denominator_unit"); ok {
		r.DenominatorUnit = v.(*apiresource.Unit)
	}
}
