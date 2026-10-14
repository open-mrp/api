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
		ObjectType: constants.ObjectTypeScanningStation,
		Load:       resourceloaders.LoadScanningStations,
		Subs: []include.SubField{
			{
				Key:         "department",
				Cardinality: include.CardinalityOnePtr,
				Populate:    populateDepartmentOnScanningStation,
			},
			{
				Key:         "production_steps",
				Cardinality: include.CardinalityList,
				Populate:    populateProductionStepsOnScanningStation,
			},
		},
	})
}

func populateDepartmentOnScanningStation(ctx context.Context, parent any, _ map[string]any) {
	ss := parent.(*apiresource.ScanningStation)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeScanningStation, ss.ID, "department")
	if !ok {
		return
	}
	ss.Department = v.(*apiresource.Department)
}

func populateProductionStepsOnScanningStation(ctx context.Context, parent any, _ map[string]any) {
	ss := parent.(*apiresource.ScanningStation)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeScanningStation, ss.ID, "production_steps")
	if !ok {
		return
	}
	steps := v.([]apiresource.ProductionStep)
	ss.ProductionSteps = apiresource.NewList(steps, apiresource.PageInfo{})
}
