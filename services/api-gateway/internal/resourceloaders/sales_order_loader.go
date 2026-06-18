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

var salesOrderLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.sales_order")

// LoadSalesOrders fetches sales orders by ID via BatchGetSalesOrdersByIDs and
// builds expandable SalesOrder references with real header data. Nested
// sub-resources (lines, addresses, customer, …) are their own expandable
// relations and are not populated here.
func LoadSalesOrders(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderLoaderTracer, "loader.sales_orders.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetSalesOrdersByIDsResponse, error) {
			return coreSalesClient.BatchGetSalesOrdersByIDs(ctx, &pb.BatchGetSalesOrdersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.SalesOrders))
	for _, so := range resp.SalesOrders {
		out[so.Id] = salesOrderReferenceFromProto(so)
	}
	return out, nil
}

func salesOrderReferenceFromProto(info *pb.SalesOrderInfo) *apiresource.SalesOrder {
	ackStatus := constants.AcknowledgmentStatusNotSent
	if info.IsAcknowledgmentSent {
		ackStatus = constants.AcknowledgmentStatusSent
	}
	return &apiresource.SalesOrder{
		ID:                          info.Id,
		Object:                      constants.ObjectTypeSalesOrder,
		Number:                      info.Number,
		CustomerPurchaseOrderNumber: info.CustomerPoNumber,
		Note:                        info.Note,
		Status:                      constants.SalesOrderStatusCode(info.StatusCode),
		Priority:                    constants.PriorityCode(info.PriorityCode),
		PaymentStatus:               constants.SalesOrderPaymentStatusUnpaid,
		AcknowledgmentStatus:        ackStatus,
		CreatedAt:                   grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:                   grpcutil.TimestampToTime(info.UpdatedAt),
	}
}
