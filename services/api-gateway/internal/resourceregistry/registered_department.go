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
		ObjectType: constants.ObjectTypeDepartment,
		Load:       resourceloaders.LoadDepartments,
		Subs: []include.SubField{
			{
				Key:         "location",
				Target:      constants.ObjectTypeLocation,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractLocationIDFromDepartment,
				Populate:    populateLocationOnDepartment,
			},
			{
				Key:         "scanning_stations",
				Cardinality: include.CardinalityList,
				Populate:    populateScanningStationsOnDepartment,
			},
			{
				Key:         "machines",
				Cardinality: include.CardinalityList,
				Populate:    populateMachinesOnDepartment,
			},
		},
	})
}

func extractLocationIDFromDepartment(ctx context.Context, parent any) []string {
	dept := parent.(*apiresource.Department)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeDepartment, dept.ID, "location_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateLocationOnDepartment(ctx context.Context, parent any, loaded map[string]any) {
	dept := parent.(*apiresource.Department)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeDepartment, dept.ID, "location_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		dept.Location = v.(*apiresource.Location)
	}
}

func populateScanningStationsOnDepartment(ctx context.Context, parent any, _ map[string]any) {
	dept := parent.(*apiresource.Department)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeDepartment, dept.ID, "scanning_stations")
	if !ok {
		return
	}
	stations := v.([]apiresource.ScanningStation)
	dept.ScanningStations = apiresource.NewList(stations, apiresource.PageInfo{})
}

func populateMachinesOnDepartment(ctx context.Context, parent any, _ map[string]any) {
	dept := parent.(*apiresource.Department)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeDepartment, dept.ID, "machines")
	if !ok {
		return
	}
	machines := v.([]apiresource.Machine)
	dept.Machines = apiresource.NewList(machines, apiresource.PageInfo{})
}
