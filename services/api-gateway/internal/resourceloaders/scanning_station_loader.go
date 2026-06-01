package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var scanningStationLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.scanning_station")

func LoadScanningStations(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, scanningStationLoaderTracer, "loader.scanning_stations.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetScanningStationsByIDsResponse, error) {
			return coreClient.BatchGetScanningStationsByIDs(ctx, &pb.BatchGetScanningStationsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.ScanningStations))
	for _, ss := range resp.ScanningStations {
		out[ss.Id] = ScanningStationFromProto(ss)

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
			meta.Set(constants.ObjectTypeScanningStation, ss.Id, "department", dept)
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
		meta.Set(constants.ObjectTypeScanningStation, ss.Id, "production_steps", steps)
	}
	return out, nil
}

func ScanningStationFromLightProto(ss *pb.LightScanningStationInfo) *apiresource.ScanningStation {
	return &apiresource.ScanningStation{
		ID:                  ss.Id,
		Object:              constants.ObjectTypeScanningStation,
		Name:                ss.Name,
		Type:                constants.ScanningStationType(ss.Type),
		OperatorRequirement: constants.OperatorRequirement(ss.OperatorRequirement),
		CreatedAt:           grpcutil.TimestampToTime(ss.CreatedAt),
		UpdatedAt:           grpcutil.TimestampToTime(ss.UpdatedAt),
	}
}

func ScanningStationFromProto(ss *pb.ScanningStationInfo) *apiresource.ScanningStation {
	station := &apiresource.ScanningStation{
		ID:                  ss.Id,
		Object:              constants.ObjectTypeScanningStation,
		Name:                ss.Name,
		Notes:               ss.Notes,
		Type:                constants.ScanningStationType(ss.Type),
		OperatorRequirement: constants.OperatorRequirement(ss.OperatorRequirement),
		CreatedAt:           grpcutil.TimestampToTime(ss.CreatedAt),
		UpdatedAt:           grpcutil.TimestampToTime(ss.UpdatedAt),
	}

	if ss.LabelSizeCode != nil {
		v := constants.LabelSizeCode(*ss.LabelSizeCode)
		station.LabelSizeCode = &v
	}
	if ss.LabelTypeCode != nil {
		v := constants.LabelTypeCode(*ss.LabelTypeCode)
		station.LabelTypeCode = &v
	}

	return station
}
