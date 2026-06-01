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
		ObjectType: constants.ObjectTypeScanningStation,
		Load:       resourceloaders.LoadScanningStations,
		Subs: []resourcekit.SubField{
			{
				Key:         "department",
				Cardinality: resourcekit.CardinalityOnePtr,
				Populate:    populateDepartmentOnScanningStation,
			},
			{
				Key:         "production_steps",
				Cardinality: resourcekit.CardinalityList,
				Populate:    populateProductionStepsOnScanningStation,
			},
		},
	})
}

func populateDepartmentOnScanningStation(ctx context.Context, parent any, _ map[string]any) {
	ss := parent.(*apiresource.ScanningStation)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeScanningStation, ss.ID, "department")
	if !ok {
		return
	}
	ss.Department = v.(*apiresource.Department)
}

func populateProductionStepsOnScanningStation(ctx context.Context, parent any, _ map[string]any) {
	ss := parent.(*apiresource.ScanningStation)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeScanningStation, ss.ID, "production_steps")
	if !ok {
		return
	}
	steps := v.([]apiresource.ProductionStep)
	ss.ProductionSteps = apiresource.NewList(steps, apiresource.PageInfo{})
}
