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

var departmentLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.department")

func LoadDepartments(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, departmentLoaderTracer, "loader.departments.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetDepartmentsByIDsResponse, error) {
			return coreClient.BatchGetDepartmentsByIDs(ctx, &pb.BatchGetDepartmentsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Departments))
	for _, d := range resp.Departments {
		out[d.Id] = departmentFromProto(d)

		var locationID string
		if d.LocationId != nil {
			locationID = *d.LocationId
		}
		meta.Set(constants.ObjectTypeDepartment, d.Id, "location_id", locationID)

		stations := make([]apiresource.ScanningStation, len(d.ScanningStations))
		for i, s := range d.ScanningStations {
			stations[i] = *ScanningStationFromLightProto(s)
		}
		meta.Set(constants.ObjectTypeDepartment, d.Id, "scanning_stations", stations)

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
		meta.Set(constants.ObjectTypeDepartment, d.Id, "machines", machines)
	}
	return out, nil
}

func departmentFromProto(d *pb.DepartmentInfo) *apiresource.Department {
	return &apiresource.Department{
		ID:        d.Id,
		Object:    constants.ObjectTypeDepartment,
		Name:      d.Name,
		Notes:     d.Notes,
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}
}
