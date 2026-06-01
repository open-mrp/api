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

var productLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.product")

func LoadProducts(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, productLoaderTracer, "loader.products.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetProductsByIDsResponse, error) {
			return coreClient.BatchGetProductsByIDs(ctx, &pb.BatchGetProductsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Products))
	for _, p := range resp.Products {
		out[p.Id] = productFromProto(p)
		meta.Set(constants.ObjectTypeProduct, p.Id, "item_id", p.ItemId)
		if p.ProductLineId != nil {
			meta.Set(constants.ObjectTypeProduct, p.Id, "product_line_id", *p.ProductLineId)
		}
	}
	return out, nil
}

func productFromProto(p *pb.ProductFullInfo) *apiresource.Product {
	res := &apiresource.Product{
		ID:        p.Id,
		Object:    constants.ObjectTypeProduct,
		Type:      constants.ProductTypeCode(p.ProductTypeCode),
		CreatedAt: grpcutil.TimestampToTime(p.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(p.UpdatedAt),
	}
	if p.IsPortalReady {
		res.PortalVisibility = constants.CustomerPortalVisibilityVisible
	} else {
		res.PortalVisibility = constants.CustomerPortalVisibilityHidden
	}
	return res
}
