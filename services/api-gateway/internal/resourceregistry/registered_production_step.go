package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeProductionStep,
		Load:       resourceloaders.LoadProductionSteps,
		Subs: []resourcekit.SubField{
			// Target + ExtractRefs (not a loader) because both are already stashed by the step
			// presenter; they exist so the resolver descends for production.produced_item and
			// consumptions.consumed_item.
			{Key: "production", Target: constants.ObjectTypeProduction, ExtractRefs: extractProductionRefFromProductionStep, Populate: populateProductionOnProductionStep},
			{Key: "consumptions", Target: constants.ObjectTypeConsumption, ExtractRefs: extractConsumptionRefsFromProductionStep, Populate: populateConsumptionsOnProductionStep},
			{Key: "machines", Populate: populateMachinesOnProductionStep},
			{
				Key:         "scanning_station",
				Target:      constants.ObjectTypeScanningStation,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractScanningStationIDFromProductionStep,
				Populate:    populateScanningStationOnProductionStep,
			},
			{
				Key:         "department",
				Target:      constants.ObjectTypeDepartment,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractDepartmentIDFromProductionStep,
				Populate:    populateDepartmentOnProductionStep,
			},
			{Key: "in_steps", Populate: populateInStepsOnProductionStep},
			{Key: "out_steps", Populate: populateOutStepsOnProductionStep},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeProduction,
		Load:       resourceloaders.LoadProductions,
		Subs: []resourcekit.SubField{
			{Key: "produced_item", Target: constants.ObjectTypeItem, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractProducedItemIDFromProduction, Populate: populateProducedItemOnProduction},
		},
	})
}

func populateProductionOnProductionStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "production")
	if !ok {
		return
	}
	ps.Production = v.(*apiresource.ProductionOutput)
}

func populateConsumptionsOnProductionStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "consumptions")
	if !ok {
		return
	}
	ps.Consumptions = v.(*apiresource.List[apiresource.Consumption])
}

func populateMachinesOnProductionStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "machines")
	if !ok {
		return
	}
	ps.Machines = v.(*apiresource.List[apiresource.Machine])
}

func extractScanningStationIDFromProductionStep(ctx context.Context, parent any) []string {
	ps := parent.(*apiresource.ProductionStep)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionStep, ps.ID, "scanning_station_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateScanningStationOnProductionStep(ctx context.Context, parent any, loaded map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionStep, ps.ID, "scanning_station_id")
	if v, ok := loaded[id]; ok {
		ps.ScanningStation = v.(*apiresource.ScanningStation)
	}
}

func extractDepartmentIDFromProductionStep(ctx context.Context, parent any) []string {
	ps := parent.(*apiresource.ProductionStep)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionStep, ps.ID, "department_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateDepartmentOnProductionStep(ctx context.Context, parent any, loaded map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionStep, ps.ID, "department_id")
	if v, ok := loaded[id]; ok {
		ps.Department = v.(*apiresource.Department)
	}
}

func populateInStepsOnProductionStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "in_steps")
	if !ok {
		return
	}
	ps.InSteps = v.(*apiresource.List[apiresource.ProductionStep])
}

func populateOutStepsOnProductionStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "out_steps")
	if !ok {
		return
	}
	ps.OutSteps = v.(*apiresource.List[apiresource.ProductionStep])
}

// The resolver runs Populate before gathering refs, so both are already on the step by the time
// these are called.
func extractProductionRefFromProductionStep(_ context.Context, parent any) []any {
	ps := parent.(*apiresource.ProductionStep)
	if ps.Production == nil {
		return nil
	}
	return []any{ps.Production}
}

func extractConsumptionRefsFromProductionStep(_ context.Context, parent any) []any {
	ps := parent.(*apiresource.ProductionStep)
	if ps.Consumptions == nil {
		return nil
	}
	refs := make([]any, len(ps.Consumptions.Data))
	for i := range ps.Consumptions.Data {
		refs[i] = &ps.Consumptions.Data[i]
	}
	return refs
}

func extractProducedItemIDFromProduction(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.ProductionOutput)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProduction, p.ID, "produced_item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateProducedItemOnProduction(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.ProductionOutput)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProduction, p.ID, "produced_item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.ProducedItem = v.(*apiresource.Item)
	}
}
