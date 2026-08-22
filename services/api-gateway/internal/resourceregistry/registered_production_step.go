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
			{Key: "production", Populate: populateProductionOnProductionStep},
			{Key: "consumptions", Populate: populateConsumptionsOnProductionStep},
			{Key: "machines", Populate: populateMachinesOnProductionStep},
			{Key: "scanning_station", Populate: populateScanningStationOnProductionStep},
			{Key: "department", Populate: populateDepartmentOnProductionStep},
			{Key: "in_steps", Populate: populateInStepsOnProductionStep},
			{Key: "out_steps", Populate: populateOutStepsOnProductionStep},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeProduction,
		Load:       resourceloaders.LoadProductions,
		Subs: []resourcekit.SubField{
			{Key: "produced_item", Populate: populateProducedItemOnProduction},
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

func populateScanningStationOnProductionStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "scanning_station")
	if !ok {
		return
	}
	ps.ScanningStation = v.(*apiresource.ScanningStation)
}

func populateDepartmentOnProductionStep(ctx context.Context, parent any, _ map[string]any) {
	ps := parent.(*apiresource.ProductionStep)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProductionStep, ps.ID, "department")
	if !ok {
		return
	}
	ps.Department = v.(*apiresource.Department)
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

func populateProducedItemOnProduction(ctx context.Context, parent any, _ map[string]any) {
	p := parent.(*apiresource.ProductionOutput)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeProduction, p.ID, "produced_item")
	if !ok {
		return
	}
	p.ProducedItem = v.(*apiresource.Item)
}
