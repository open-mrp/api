package resourceloaders

import (
	"context"
	"net/url"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
	"google.golang.org/grpc"
)

var workCenterLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.work_center")

// LoadWorkCenters batch-loads work centers. Each carries its location and the IDs of its first machines and scanning
// stations, newest first, for the includes that expand them.
func LoadWorkCenters(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, workCenterLoaderTracer, "loader.work_centers.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetDepartmentsByIDsResponse, error) {
			return coreClient.BatchGetDepartmentsByIDs(ctx, &pb.BatchGetDepartmentsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Departments))
	for _, d := range resp.Departments {
		out[d.Id] = workCenterFromProto(d)
		meta.Set(constants.ObjectTypeWorkCenter, d.Id, "location_id", d.GetLocationId())

		stationIDs := make([]string, len(d.ScanningStations))
		for i, s := range d.ScanningStations {
			stationIDs[i] = s.Id
		}
		meta.Set(constants.ObjectTypeWorkCenter, d.Id, "scanning_station_ids", stationIDs)

		machineIDs := make([]string, len(d.Machines))
		for i, m := range d.Machines {
			machineIDs[i] = m.Id
		}
		meta.Set(constants.ObjectTypeWorkCenter, d.Id, "machine_ids", machineIDs)
	}
	return out, nil
}

// WorkCenterMembersListPath is the machine or scanning station list narrowed to one work center, where its embedded
// members continue. resource is "machines" or "scanning-stations".
func WorkCenterMembersListPath(resource, workCenterID string) string {
	return "/v1/operations/" + resource + "?" + url.Values{"work_center_ids[]": {workCenterID}}.Encode()
}

func workCenterFromProto(d *pb.DepartmentInfo) *apiresource.WorkCenter {
	return &apiresource.WorkCenter{
		ID:        d.Id,
		Object:    constants.ObjectTypeWorkCenter,
		Name:      d.Name,
		LaborRate: UnitPriceFromProto(d.LaborRate),
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}
}
