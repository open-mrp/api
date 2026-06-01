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

var partLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.part")

func LoadParts(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, partLoaderTracer, "loader.parts.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPartsByIDsResponse, error) {
			return coreClient.BatchGetPartsByIDs(ctx, &pb.BatchGetPartsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Parts))
	for _, p := range resp.Parts {
		out[p.Id] = partFromProto(p)
		meta.Set(constants.ObjectTypePart, p.Id, "item_id", p.ItemId)
	}
	return out, nil
}

func partFromProto(p *pb.PartInfo) *apiresource.Part {
	return &apiresource.Part{
		ID:        p.Id,
		Object:    constants.ObjectTypePart,
		CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
	}
}
