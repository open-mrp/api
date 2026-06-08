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

var purchaseOrderLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.purchase_order")

// LoadPurchaseOrders fetches purchase orders by ID via BatchGetPurchaseOrdersByIDs
// and builds expandable PurchaseOrder references with real header data.
// Nested sub-resources (lines, addresses, supplier, …) are their own expandable
// relations and are not populated here.
func LoadPurchaseOrders(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, purchaseOrderLoaderTracer, "loader.purchase_orders.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetPurchaseOrdersByIDsResponse, error) {
			return corePurchaseClient.BatchGetPurchaseOrdersByIDs(ctx, &pb.BatchGetPurchaseOrdersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.PurchaseOrders))
	for _, po := range resp.PurchaseOrders {
		out[po.Id] = purchaseOrderReferenceFromProto(po)
	}
	return out, nil
}

func purchaseOrderReferenceFromProto(info *pb.PurchaseOrderInfo) *apiresource.PurchaseOrder {
	createdAt := grpcutil.TimestampToTime(info.CreatedAt)
	ackStatus := constants.AcknowledgmentStatusNotSent
	if info.IsAcknowledgmentSent {
		ackStatus = constants.AcknowledgmentStatusSent
	}
	return &apiresource.PurchaseOrder{
		ID:                   info.Id,
		Object:               constants.ObjectTypePurchaseOrder,
		Number:               info.Number,
		Note:                 info.Note,
		Status:               constants.SalesOrderStatusCode(info.StatusCode),
		Priority:             constants.PriorityCode(info.PriorityCode),
		AcknowledgmentStatus: ackStatus,
		CreatedAt:            createdAt,
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
	}
}
