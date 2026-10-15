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
				Key:         "work_center",
				Target:      constants.ObjectTypeWorkCenter,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs: func(ctx context.Context, parent any) []string {
					if id := scanningStationWorkCenterID(ctx, parent.(*apiresource.ScanningStation)); id != "" {
						return []string{id}
					}
					return nil
				},
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					ss := parent.(*apiresource.ScanningStation)
					if v, ok := loaded[scanningStationWorkCenterID(ctx, ss)]; ok {
						ss.WorkCenter = v.(*apiresource.WorkCenter)
					}
				},
			},
			{
				Key:         "production_steps",
				Cardinality: include.CardinalityList,
				Populate:    populateProductionStepsOnScanningStation,
			},
		},
	})
}

func scanningStationWorkCenterID(ctx context.Context, ss *apiresource.ScanningStation) string {
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeScanningStation, ss.ID, "work_center_id")
	return id
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
