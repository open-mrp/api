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

var orderDiscountLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.order_discount")

// LoadOrderDiscounts fetches order discounts by ID via
// BatchGetOrderDiscountsByIDs (CoreSalesService). Pure leaf — no expandable
// sub-resources, so no LoadMeta is needed.
func LoadOrderDiscounts(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountLoaderTracer, "loader.order_discounts.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetOrderDiscountsByIDsResponse, error) {
			return coreSalesClient.BatchGetOrderDiscountsByIDs(ctx, &pb.BatchGetOrderDiscountsByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.OrderDiscounts))
	for _, d := range resp.OrderDiscounts {
		out[d.Id] = OrderDiscountFromProto(d)
	}
	return out, nil
}

func OrderDiscountFromProto(d *pb.OrderDiscountInfo) *apiresource.OrderDiscount {
	return &apiresource.OrderDiscount{
		ID:           d.Id,
		Object:       constants.ObjectTypeOrderDiscount,
		Name:         d.Name,
		Code:         d.Code,
		Percentage:   d.Percentage,
		Amount:       d.Amount,
		DiscountType: constants.OrderDiscountType(d.DiscountType),
		OrderCount:   d.OrderCount,
		CreatedAt:    grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:    grpcutil.TimestampToTime(d.UpdatedAt),
	}
}
