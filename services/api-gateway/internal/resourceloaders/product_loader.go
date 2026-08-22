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

	out := make(map[string]any, len(resp.Products))
	for _, p := range resp.Products {
		out[p.Id] = ProductFromProto(p)
		StashProductMeta(ctx, p)
	}
	return out, nil
}

// StashProductMeta records the ids needed to populate a product's expandable fields (item, product_line) when the include resolver runs. Pair it with ProductFromProto so includes work without leaking the nested resources.
func StashProductMeta(ctx context.Context, p *pb.ProductFullInfo) {
	if p == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeProduct, p.Id, "item_id", p.ItemId)
	if p.ProductLineId != nil {
		meta.Set(constants.ObjectTypeProduct, p.Id, "product_line_id", *p.ProductLineId)
	}
}

// ProductFromProto builds the gated Product resource: expandable fields (item, product_line) are left nil and populated only when explicitly requested via the include resolver. Use this — never the full ProductPresenter — when building a JSON API response.
func ProductFromProto(p *pb.ProductFullInfo) *apiresource.Product {
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
