package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
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

	out := make(map[string]any, len(resp.Parts))
	for _, p := range resp.Parts {
		out[p.Id] = PartFromProto(p)
		StashPartMeta(ctx, p)
	}
	return out, nil
}

// StashPartMeta records the id needed to populate a part's expandable item when the include resolver runs. Pair it with PartFromProto so includes work without leaking the nested item.
func StashPartMeta(ctx context.Context, p *pb.PartInfo) {
	if p == nil {
		return
	}
	resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypePart, p.Id, "item_id", p.ItemId)
}

// PartFromProto builds the gated Part resource: the expandable item is left nil and populated only when explicitly requested via the include resolver. Use this — never the full PartPresenter — when building a JSON API response.
func PartFromProto(p *pb.PartInfo) *apiresource.Part {
	return &apiresource.Part{
		ID:        p.Id,
		Object:    constants.ObjectTypePart,
		CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
	}
}
