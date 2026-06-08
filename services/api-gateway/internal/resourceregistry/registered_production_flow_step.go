package resourceregistry

import (
	"context"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

const productionFlowStepType constants.ObjectType = "_production_flow_step"
const productionFlowProductionType constants.ObjectType = "_production_flow_production"
const productionFlowConsumptionType constants.ObjectType = "_production_flow_consumption"

func stubLoader(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError("stub loader should not be called")
}

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: productionFlowStepType,
		Load:       stubLoader,
		Subs: []resourcekit.SubField{
			{Key: "production", Target: productionFlowProductionType, Populate: populateProductionOnFlowStep, ExtractRefs: extractProductionRefsFromFlowStep},
			{Key: "consumptions", Target: productionFlowConsumptionType, Populate: populateConsumptionsOnFlowStep, ExtractRefs: extractConsumptionRefsFromFlowStep},
			{Key: "machines", Populate: populateMachinesOnFlowStep},
			{Key: "scanning_station", Target: constants.ObjectTypeScanningStation, ExtractIDs: extractScanningStationIDFromFlowStep, Populate: populateScanningStationOnFlowStep},
			{Key: "department", Target: constants.ObjectTypeDepartment, ExtractIDs: extractDepartmentIDFromFlowStep, Populate: populateDepartmentOnFlowStep},
			{Key: "in_steps", Populate: populateInStepsOnFlowStep},
			{Key: "out_steps", Populate: populateOutStepsOnFlowStep},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: productionFlowProductionType,
		Load:       stubLoader,
		Subs: []resourcekit.SubField{
			{Key: "produced_item", Target: constants.ObjectTypeItem, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractProducedItemIDFromFlowProduction, Populate: populateProducedItemOnFlowProduction},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: productionFlowConsumptionType,
		Load:       stubLoader,
		Subs: []resourcekit.SubField{
			{Key: "consumed_item", Target: constants.ObjectTypeItem, Cardinality: resourcekit.CardinalityOnePtr, ExtractIDs: extractConsumedItemIDFromFlowConsumption, Populate: populateConsumedItemOnFlowConsumption},
			{Key: "quantity", Populate: populateQuantityOnFlowConsumption},
			{Key: "waste_quantity", Populate: populateWasteQuantityOnFlowConsumption},
		},
	})
}

func extractProductionRefsFromFlowStep(_ context.Context, parent any) []any {
	ps := parent.(*apiresource.ProductionFlowStep)
	if ps.Production == nil {
		return nil
	}
	return []any{ps.Production}
}

func extractConsumptionRefsFromFlowStep(_ context.Context, parent any) []any {
	ps := parent.(*apiresource.ProductionFlowStep)
	if ps.Consumptions == nil {
		return nil
	}
	refs := make([]any, len(ps.Consumptions.Data))
	for i := range ps.Consumptions.Data {
		refs[i] = &ps.Consumptions.Data[i]
	}
	return refs
}

func populateProductionOnFlowStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionFlowStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "production")
	if !ok {
		return
	}
	ps.Production = v.(*apiresource.ProductionFlowProduction)
}

func populateConsumptionsOnFlowStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionFlowStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "consumptions")
	if !ok {
		return
	}
	ps.Consumptions = v.(*apiresource.List[apiresource.ProductionFlowConsumption])
}

func populateMachinesOnFlowStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionFlowStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "machines")
	if !ok {
		return
	}
	ps.Machines = v.(*apiresource.List[apiresource.Machine])
}

func extractScanningStationIDFromFlowStep(ctx context.Context, parent any) []string {
	ps := parent.(*apiresource.ProductionFlowStep)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionStep, ps.ID, "scanning_station_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateScanningStationOnFlowStep(ctx context.Context, parent any, loaded map[string]any) {
	ps := parent.(*apiresource.ProductionFlowStep)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionStep, ps.ID, "scanning_station_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ps.ScanningStation = v.(*apiresource.ScanningStation)
	}
}

func extractDepartmentIDFromFlowStep(ctx context.Context, parent any) []string {
	ps := parent.(*apiresource.ProductionFlowStep)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionStep, ps.ID, "department_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateDepartmentOnFlowStep(ctx context.Context, parent any, loaded map[string]any) {
	ps := parent.(*apiresource.ProductionFlowStep)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProductionStep, ps.ID, "department_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ps.Department = v.(*apiresource.Department)
	}
}

func populateInStepsOnFlowStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionFlowStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "in_steps")
	if !ok {
		return
	}
	ps.InSteps = v.(*apiresource.List[apiresource.ProductionStep])
}

func populateOutStepsOnFlowStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionFlowStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "out_steps")
	if !ok {
		return
	}
	ps.OutSteps = v.(*apiresource.List[apiresource.ProductionStep])
}

func extractProducedItemIDFromFlowProduction(ctx context.Context, parent any) []string {
	p := parent.(*apiresource.ProductionFlowProduction)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProduction, p.ID, "produced_item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateProducedItemOnFlowProduction(ctx context.Context, parent any, loaded map[string]any) {
	p := parent.(*apiresource.ProductionFlowProduction)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeProduction, p.ID, "produced_item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		p.ProducedItem = v.(*apiresource.Item)
	}
}

func extractConsumedItemIDFromFlowConsumption(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.ProductionFlowConsumption)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeConsumption, c.ID, "consumed_item_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateConsumedItemOnFlowConsumption(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.ProductionFlowConsumption)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeConsumption, c.ID, "consumed_item_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		c.ConsumedItem = v.(*apiresource.Item)
	}
}

func populateQuantityOnFlowConsumption(_ context.Context, _ any, _ map[string]any) {
	// quantity is always present on consumptions, not expandable
}

func populateWasteQuantityOnFlowConsumption(_ context.Context, _ any, _ map[string]any) {
	// waste_quantity is always present on consumptions, not expandable
}
