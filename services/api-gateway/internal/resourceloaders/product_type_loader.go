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

var productTypeLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.product_type")

// LoadProductTypes fetches product types by ID via BatchGetProductTypesByIDs. ProductType is a system-only lookup with no expandable sub-resources, so no LoadMeta is required.
func LoadProductTypes(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, productTypeLoaderTracer, "loader.product_types.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetProductTypesByIDsResponse, error) {
			return coreClient.BatchGetProductTypesByIDs(ctx, &pb.BatchGetProductTypesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.ProductTypes))
	for _, pt := range resp.ProductTypes {
		out[pt.Id] = productTypeFromProto(pt)
	}
	return out, nil
}

func productTypeFromProto(pt *pb.ProductTypeInfo) *apiresource.ProductType {
	return &apiresource.ProductType{
		ID:        pt.Id,
		Object:    constants.ObjectTypeProductType,
		Name:      pt.Name,
		Code:      constants.ProductTypeCode(pt.Code),
		CreatedAt: grpcutil.TimestampToTime(pt.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(pt.UpdatedAt),
	}
}
