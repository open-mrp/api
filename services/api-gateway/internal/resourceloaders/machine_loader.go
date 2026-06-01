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
	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Machines))
	for _, m := range resp.Machines {
		out[m.Id] = MachineFromProto(m)

		var departmentID string
		if m.DepartmentId != nil {
			departmentID = *m.DepartmentId
		}
		meta.Set(constants.ObjectTypeMachine, m.Id, "department_id", departmentID)
	}
	return out, nil
}

func MachineFromProto(m *pb.MachineInfo) *apiresource.Machine {
	return &apiresource.Machine{
		ID:           m.Id,
		Object:       constants.ObjectTypeMachine,
		Name:         m.Name,
		SerialNumber: m.SerialNumber,
		Notes:        m.Notes,
		CreatedAt:    grpcutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(m.UpdatedAt),
	}
}
