package orderdiscountep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type OrderDiscountSvc interface {
	ListOrderDiscounts(ctx context.Context, req *ListOrderDiscountsRequest) (*apiresource.List[apiresource.OrderDiscount], *apierror.APIError)
	GetOrderDiscount(ctx context.Context, req *RetrieveOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError)
	CreateOrderDiscount(ctx context.Context, req *CreateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError)
	UpdateOrderDiscount(ctx context.Context, req *UpdateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError)
	DeleteOrderDiscount(ctx context.Context, req *DeleteOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError)
	FindOrderDiscountByCode(ctx context.Context, req *FindOrderDiscountByCodeRequest) (*apiresource.OrderDiscount, *apierror.APIError)
}

type OrderDiscountSvcConfig struct {
	CoreClient pb.CoreSalesServiceClient
}

type orderDiscountSvcImpl struct {
	coreClient pb.CoreSalesServiceClient
}

var orderDiscountSvcTracer = tracing.GetTracer("api-gateway.endpoints.order-discounts.service")

func (c *OrderDiscountSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("order discount endpoint service: core client is required")
	}
	return nil
}

func NewOrderDiscountSvc(config *OrderDiscountSvcConfig) OrderDiscountSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &orderDiscountSvcImpl{coreClient: config.CoreClient}
}

func (m *orderDiscountSvcImpl) ListOrderDiscounts(ctx context.Context, req *ListOrderDiscountsRequest) (*apiresource.List[apiresource.OrderDiscount], *apierror.APIError) {
	pbReq := &pb.ListOrderDiscountsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListOrderDiscountsResponse, error) {
			return m.coreClient.ListOrderDiscounts(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.OrderDiscounts))
	for i, d := range resp.OrderDiscounts {
		ids[i] = d.Id
	}
	loaded, apiErr := resourceloaders.LoadOrderDiscounts(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.OrderDiscount, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.OrderDiscount)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *orderDiscountSvcImpl) GetOrderDiscount(ctx context.Context, req *RetrieveOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	return loadOrderDiscountByID(ctx, req.OrderDiscountID)
}

func (m *orderDiscountSvcImpl) CreateOrderDiscount(ctx context.Context, req *CreateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	pbReq := &pb.CreateOrderDiscountRequest{
		Name:         req.Name,
		Code:         req.Code,
		Percentage:   req.Percentage.Ptr(),
		Amount:       req.Amount.Ptr(),
		DiscountType: req.DiscountType,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateOrderDiscountResponse, error) {
			return m.coreClient.CreateOrderDiscount(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadOrderDiscountByID(ctx, resp.OrderDiscount.Id)
}

func (m *orderDiscountSvcImpl) UpdateOrderDiscount(ctx context.Context, req *UpdateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	pbReq := &pb.UpdateOrderDiscountRequest{
		Id:           req.OrderDiscountID,
		Name:         req.Name.Ptr(),
		Code:         req.Code.Ptr(),
		Percentage:   req.Percentage.Ptr(),
		Amount:       req.Amount.Ptr(),
		DiscountType: req.DiscountType.Ptr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateOrderDiscountResponse, error) {
			return m.coreClient.UpdateOrderDiscount(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadOrderDiscountByID(ctx, resp.OrderDiscount.Id)
}

func (m *orderDiscountSvcImpl) DeleteOrderDiscount(ctx context.Context, req *DeleteOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	// Delete returns the deleted resource. The legacy DeleteOrderDiscount RPC
	// returns the resource pre-delete, so we map directly from its response
	// rather than running through LoadOrderDiscounts (the row no longer exists).
	pbReq := &pb.DeleteOrderDiscountRequest{Id: req.OrderDiscountID}
	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteOrderDiscountResponse, error) {
			return m.coreClient.DeleteOrderDiscount(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return resourceloaders.OrderDiscountFromProto(resp.OrderDiscount), nil
}

func (m *orderDiscountSvcImpl) FindOrderDiscountByCode(ctx context.Context, req *FindOrderDiscountByCodeRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	pbReq := &pb.FindOrderDiscountByCodeRequest{
		Code:           req.Code,
		BuyerAccountId: req.BuyerAccountID.Ptr(),
		SalesOrderId:   req.SalesOrderID.Ptr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.find_by_code", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.FindOrderDiscountByCodeResponse, error) {
			return m.coreClient.FindOrderDiscountByCode(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadOrderDiscountByID(ctx, resp.OrderDiscount.Id)
}

// loadOrderDiscountByID wraps the single-ID load pattern used after mutations
// and for the retrieve / find-by-code endpoints.
func loadOrderDiscountByID(ctx context.Context, id string) (*apiresource.OrderDiscount, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadOrderDiscounts(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Order discount not found.")
	}
	return v.(*apiresource.OrderDiscount), nil
}
