package salesorderstatusep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type SalesOrderStatusSvc interface {
	ListSalesOrderStatuses(ctx context.Context, req *ListSalesOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError)
}

type SalesOrderStatusSvcConfig struct {
	// CoreClient (required) is the core-service sales gRPC client.
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
	return &salesOrderStatusSvcImpl{coreClient: config.CoreClient}
}

func (m *salesOrderStatusSvcImpl) ListSalesOrderStatuses(ctx context.Context, req *ListSalesOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError) {
	pbReq := &pb.ListSalesOrderStatusesRequest{Cursor: req.Cursor, Limit: req.Limit, Query: req.Query}
	resp, apiErr := grpcutil.CallRPC(ctx, salesOrderStatusSvcTracer, "service.sales_order_statuses.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListSalesOrderStatusesResponse, error) {
			return m.coreClient.ListSalesOrderStatuses(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.SalesOrderStatuses))
	for i, s := range resp.SalesOrderStatuses {
		ids[i] = s.Id
	}
	loaded, apiErr := resourceloaders.LoadSalesOrderStatuses(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.SalesOrderStatus, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.SalesOrderStatus)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}
