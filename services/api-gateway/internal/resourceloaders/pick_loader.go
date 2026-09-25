package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var pickLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.pick")

// LoadPicks builds expandable Pick references with real header data. Nested sub-resources (lines,
// sales_order, customer) are their own expandable relations and are not populated here. A pick
// deleted since the reference was read (deleting a shipment deletes its pick) is simply absent, so
// its row's related.pick stays null rather than failing the page.
func LoadPicks(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, pickLoaderTracer, "loader.picks.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPicksByIDsResponse, error) {
			return corePickingClient.BatchGetPicksByIDs(ctx, &pb.BatchGetPicksByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		if omitOnUnauthorized(apiErr) {
			return map[string]any{}, nil
		}
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Picks))
	for _, pick := range resp.Picks {
		out[pick.Id] = pickReferenceFromProto(pick)
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

// Satisfies Register's required Load but is never invoked: pick lines are expandable only by
// traversal from their pick (attached inline via ExtractRefs), never loaded by their own id.
func LoadPickLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPickLines must not be called — pick lines are attached by the pick and traversed via ExtractRefs, not loaded by id",
	)
}
