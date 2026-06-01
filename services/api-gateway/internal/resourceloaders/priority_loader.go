package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var priorityLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.priority")

// LoadPriorities fetches priorities by ID via BatchGetPrioritiesByIDs. Priority
// is a system-only resource — Owner is always SystemOwner — so no FK metadata
// needs to be stashed in LoadMeta.
func LoadPriorities(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, priorityLoaderTracer, "loader.priorities.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPrioritiesByIDsResponse, error) {
			return coreClient.BatchGetPrioritiesByIDs(ctx, &pb.BatchGetPrioritiesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Priorities))
	for _, p := range resp.Priorities {
		out[p.Id] = priorityFromProto(p)
	}
	return out, nil
}

func priorityFromProto(p *pb.PriorityInfo) *apiresource.Priority {
	return &apiresource.Priority{
		ID:        p.Id,
		Object:    constants.ObjectTypePriority,
		Name:      p.Name,
		Code:      constants.PriorityCode(p.Code),
		CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
	}
}
