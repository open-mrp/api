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

var salesOrderStatusLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.sales_order_status")

func LoadSalesOrderStatuses(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderStatusLoaderTracer, "loader.sales_order_statuses.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetSalesOrderStatusesByIDsResponse, error) {
			return coreSalesClient.BatchGetSalesOrderStatusesByIDs(ctx, &pb.BatchGetSalesOrderStatusesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.SalesOrderStatuses))
	for _, s := range resp.SalesOrderStatuses {
		out[s.Id] = &apiresource.SalesOrderStatus{
			ID:        s.Id,
			Object:    constants.ObjectTypeSalesOrderStatus,
			Code:      constants.SalesOrderStatusCode(s.Code),
			Name:      s.Name,
			CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
		}
	}
	return out, nil
}
