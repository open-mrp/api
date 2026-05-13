package departmentep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func DepartmentPresenter(d *pb.DepartmentInfo) apiresource.Department {
	if d == nil {
		return apiresource.Department{}
	}

	dept := apiresource.Department{
		ID:        d.Id,
		Object:    constants.ObjectTypeDepartment,
		Name:      d.Name,
		Notes:     d.Notes,
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}

	if d.LocationId != nil && *d.LocationId != "" {
		locationName := ""
		if d.LocationName != nil {
			locationName = *d.LocationName
		}
		var typeCode constants.LocationTypeCode
		if d.LocationTypeCode != nil {
			typeCode = constants.LocationTypeCode(*d.LocationTypeCode)
		}
		dept.Location = &apiresource.Location{
			ID:       *d.LocationId,
			Object:   constants.ObjectTypeLocation,
			Name:     locationName,
			TypeCode: typeCode,
		}
	}

	if len(d.ScanningStations) > 0 {
		stations := make([]apiresource.ScanningStation, len(d.ScanningStations))
		for i, s := range d.ScanningStations {
			stations[i] = apiresource.ScanningStation{
				ID:        s.Id,
				Object:    constants.ObjectTypeScanningStation,
				Name:      s.Name,
				Type:      constants.ScanningStationType(s.Type),
				CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
				UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
			}
		}
		dept.ScanningStations = apiresource.NewList(stations, apiresource.PageInfo{})
	}

	if len(d.Machines) > 0 {
		machines := make([]apiresource.Machine, len(d.Machines))
		for i, m := range d.Machines {
			machines[i] = apiresource.Machine{
				ID:           m.Id,
				Object:       constants.ObjectTypeMachine,
				Name:         m.Name,
				SerialNumber: m.SerialNumber,
				CreatedAt:    grpcutil.TimestampToTime(m.CreatedAt),
				UpdatedAt:    grpcutil.TimestampToTime(m.UpdatedAt),
			}
		}
		dept.Machines = apiresource.NewList(machines, apiresource.PageInfo{})
	}

	return dept
}

func DepartmentListPresenter(resp *pb.ListDepartmentsResponse) *apiresource.List[apiresource.Department] {
	if resp == nil {
		return apiresource.NewList[apiresource.Department](nil, apiresource.PageInfo{})
	}

	depts := make([]apiresource.Department, len(resp.Departments))
	for i, d := range resp.Departments {
		depts[i] = DepartmentPresenter(d)
	}

	return apiresource.NewList(depts, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
