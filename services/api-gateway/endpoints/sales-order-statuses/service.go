package salesorderstatusep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type SalesOrderStatusSvc interface {
	ListSalesOrderStatuses(ctx context.Context, req *ListSalesOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError)
}

type SalesOrderStatusSvcConfig struct {
	CoreClient pb.CoreSalesServiceClient
}

type salesOrderStatusSvcImpl struct {
	coreClient pb.CoreSalesServiceClient
}

var salesOrderStatusSvcTracer = tracing.GetTracer("api-gateway.endpoints.sales-order-statuses.service")

func (c *SalesOrderStatusSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sales order status endpoint service: core client is required")
	}
	return nil
}

func NewSalesOrderStatusSvc(config *SalesOrderStatusSvcConfig) SalesOrderStatusSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &salesOrderStatusSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *salesOrderStatusSvcImpl) ListSalesOrderStatuses(ctx context.Context, req *ListSalesOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError) {
	pbReq := &pb.ListSalesOrderStatusesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderStatusSvcTracer, "service.sales_order_statuses.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesOrderStatusesResponse, error) {
			return m.coreClient.ListSalesOrderStatuses(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return SalesOrderStatusListPresenter(resp), nil
}
