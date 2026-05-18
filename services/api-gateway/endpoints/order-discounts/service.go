package orderdiscountep

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

	return &orderDiscountSvcImpl{
		coreClient: config.CoreClient,
	}
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

	return OrderDiscountListPresenter(ctx, resp), nil
}

func (m *orderDiscountSvcImpl) GetOrderDiscount(ctx context.Context, req *RetrieveOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	pbReq := &pb.GetOrderDiscountRequest{
		Id: req.OrderDiscountID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetOrderDiscountResponse, error) {
			return m.coreClient.GetOrderDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := OrderDiscountPresenter(resp.OrderDiscount)
	return &result, nil
}

func (m *orderDiscountSvcImpl) CreateOrderDiscount(ctx context.Context, req *CreateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	pbReq := &pb.CreateOrderDiscountRequest{
		Name:         req.Name,
		Code:         req.Code,
		Percentage:   req.Percentage,
		Amount:       req.Amount,
		DiscountType: req.DiscountType,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateOrderDiscountResponse, error) {
			return m.coreClient.CreateOrderDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := OrderDiscountPresenter(resp.OrderDiscount)
	return &result, nil
}

func (m *orderDiscountSvcImpl) UpdateOrderDiscount(ctx context.Context, req *UpdateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	pbReq := &pb.UpdateOrderDiscountRequest{
		Id:           req.OrderDiscountID,
		Name:         req.Name,
		Code:         req.Code,
		Percentage:   req.Percentage,
		Amount:       req.Amount,
		DiscountType: req.DiscountType,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateOrderDiscountResponse, error) {
			return m.coreClient.UpdateOrderDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := OrderDiscountPresenter(resp.OrderDiscount)
	return &result, nil
}

func (m *orderDiscountSvcImpl) DeleteOrderDiscount(ctx context.Context, req *DeleteOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	pbReq := &pb.DeleteOrderDiscountRequest{
		Id: req.OrderDiscountID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteOrderDiscountResponse, error) {
			return m.coreClient.DeleteOrderDiscount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := OrderDiscountPresenter(resp.OrderDiscount)
	return &result, nil
}

func (m *orderDiscountSvcImpl) FindOrderDiscountByCode(ctx context.Context, req *FindOrderDiscountByCodeRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
	pbReq := &pb.FindOrderDiscountByCodeRequest{
		Code:           req.Code,
		BuyerAccountId: req.BuyerAccountID,
		SalesOrderId:   req.SalesOrderID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, orderDiscountSvcTracer, "service.order_discounts.find_by_code", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.FindOrderDiscountByCodeResponse, error) {
			return m.coreClient.FindOrderDiscountByCode(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := OrderDiscountPresenter(resp.OrderDiscount)
	return &result, nil
}
