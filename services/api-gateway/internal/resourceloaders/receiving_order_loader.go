package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/safeconv"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var receivingOrderLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.receiving_order")

// LoadReceivingOrders fetches receiving orders by ID via GetReceivingOrder and builds expandable ReceivingOrder references with real header data. There is no batch RPC for receiving orders, so each ID is fetched individually. Nested sub-resources (lines, supplier, purchase_order) are their own expandable relations and are not populated here.
func LoadReceivingOrders(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := make(map[string]any, len(ids))
	for _, id := range ids {
		resp, apiErr := grpcutil.CallRPC(ctx, receivingOrderLoaderTracer, "loader.receiving_orders.get", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetReceivingOrderResponse, error) {
				return coreReceivingClient.GetReceivingOrder(ctx, &pb.GetReceivingOrderRequest{Id: id}, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}
		if resp.ReceivingOrder == nil {
			continue
		}
		out[resp.ReceivingOrder.Id] = receivingOrderReferenceFromProto(resp.ReceivingOrder)
	}
	return out, nil
}

func receivingOrderReferenceFromProto(info *pb.ReceivingOrderInfo) *apiresource.ReceivingOrder {
	return &apiresource.ReceivingOrder{
		ID:          info.Id,
		Object:      constants.ObjectTypeReceivingOrder,
		Number:      info.Number,
		Note:        info.Note,
		LineCount:   safeconv.IntToInt32(len(info.Lines)),
		CompletedAt: grpcutil.TimestampToTimePtr(info.CompletedAt),
		CreatedAt:   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:   grpcutil.TimestampToTime(info.UpdatedAt),
	}
}
