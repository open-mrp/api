package resourceloaders

import (
	"context"
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

var machineLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.machine")

func LoadMachines(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, machineLoaderTracer, "loader.machines.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetMachinesByIDsResponse, error) {
			return fulfillmentClient.BatchGetMachinesByIDs(ctx, &pb.BatchGetMachinesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	meta := include.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Machines))
	for _, m := range resp.Machines {
		out[m.Id] = MachineFromProto(m)

		meta.Set(constants.ObjectTypeMachine, m.Id, "work_center_id", m.GetDepartmentId())
	}
	return out, nil
}

func MachineFromProto(m *pb.MachineInfo) *apiresource.Machine {
	return &apiresource.Machine{
		ID:           m.Id,
		Object:       constants.ObjectTypeMachine,
		Name:         m.Name,
		SerialNumber: m.SerialNumber,
		Status:       constants.ArchiveStatus(m.Status),
		CreatedAt:    grpcutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(m.UpdatedAt),
	}
}
