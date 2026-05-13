package scanningstationep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ScanningStationPresenter(ss *pb.ScanningStationInfo) apiresource.ScanningStation {
	if ss == nil {
		return apiresource.ScanningStation{}
	}

	var labelSizeCode *constants.LabelSizeCode
	if ss.LabelSizeCode != nil {
		v := constants.LabelSizeCode(*ss.LabelSizeCode)
		labelSizeCode = &v
	}

	var labelTypeCode *constants.LabelTypeCode
	if ss.LabelTypeCode != nil {
		v := constants.LabelTypeCode(*ss.LabelTypeCode)
		labelTypeCode = &v
	}

	station := apiresource.ScanningStation{
		ID:                  ss.Id,
		Object:              constants.ObjectTypeScanningStation,
		Name:                ss.Name,
		Notes:               ss.Notes,
		Type:                constants.ScanningStationType(ss.Type),
		LabelSizeCode:       labelSizeCode,
		LabelTypeCode:       labelTypeCode,
		OperatorRequirement: constants.OperatorRequirement(ss.OperatorRequirement),
		CreatedAt:           grpcutil.TimestampToTime(ss.CreatedAt),
		UpdatedAt:           grpcutil.TimestampToTime(ss.UpdatedAt),
	}

	if ss.DepartmentId != "" {
		dept := &apiresource.Department{
			ID:     ss.DepartmentId,
			Object: constants.ObjectTypeDepartment,
			Name:   ss.DepartmentName,
		}
		if ss.DepartmentCreatedAt != nil {
			dept.CreatedAt = ss.DepartmentCreatedAt.AsTime()
		}
		if ss.DepartmentUpdatedAt != nil {
			dept.UpdatedAt = ss.DepartmentUpdatedAt.AsTime()
		}
		station.Department = dept
	}

	steps := make([]apiresource.ProductionStep, len(ss.ProductionSteps))
	for i, s := range ss.ProductionSteps {
		steps[i] = apiresource.ProductionStep{
			ID:             s.Id,
			Object:         constants.ObjectTypeProductionStep,
			Name:           s.Name,
			LevelingFactor: s.GetLevelingFactor(),
			Allowances:     s.GetAllowances(),
		}
		if s.CreatedAt != nil {
			steps[i].CreatedAt = s.CreatedAt.AsTime()
		}
		if s.UpdatedAt != nil {
			steps[i].UpdatedAt = s.UpdatedAt.AsTime()
		}
	}
	station.ProductionSteps = apiresource.NewList(steps, apiresource.PageInfo{})

	return station
}

func ScanningStationListPresenter(resp *pb.ListScanningStationsResponse) *apiresource.List[apiresource.ScanningStation] {
	if resp == nil {
		return apiresource.NewList[apiresource.ScanningStation](nil, apiresource.PageInfo{})
	}

	stations := make([]apiresource.ScanningStation, len(resp.ScanningStations))
	for i, ss := range resp.ScanningStations {
		stations[i] = ScanningStationPresenter(ss)
	}

	return apiresource.NewList(stations, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
