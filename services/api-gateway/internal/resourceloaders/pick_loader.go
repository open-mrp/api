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

var pickLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.pick")

// LoadPicks fetches picks by ID via GetPick and builds expandable Pick
// references with real header data. There is no batch RPC for picks, so each ID
// is fetched individually. Nested sub-resources (lines, sales_order, customer,
// departments) are their own expandable relations and are not populated here.
func LoadPicks(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, pickLoaderTracer, "loader.picks.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPickResponse, error) {
				return corePickingClient.GetPick(ctx, &pb.GetPickRequest{Id: id}, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}
		if resp.Pick == nil {
			continue
		}
		out[resp.Pick.Id] = pickReferenceFromProto(resp.Pick)
	}
	return out, nil
}

func pickReferenceFromProto(info *pb.PickInfo) *apiresource.Pick {
	return &apiresource.Pick{
		ID:         info.Id,
		Object:     constants.ObjectTypePick,
		Number:     info.Number,
		Priority:   constants.PriorityCode(info.PriorityCode),
		FinishedAt: grpcutil.TimestampToTimePtr(info.FinishedAt),
		CreatedAt:  grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:  grpcutil.TimestampToTime(info.UpdatedAt),
	}
}

func LoadPickLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPickLines should not be called — pick lines are not used as expandable sub-resources",
	)
}
