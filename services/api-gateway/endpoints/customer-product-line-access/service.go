package customerproductlineaccessep

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
	"google.golang.org/protobuf/types/known/emptypb"
)

type CustomerProductLineAccessSvc interface {
	ListCustomerProductLineAccess(ctx context.Context, req *ListCustomerProductLineAccessRequest) (*apiresource.List[apiresource.CustomerProductLineAccess], *apierror.APIError)
	GetCustomerProductLineAccess(ctx context.Context, req *GetCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError)
	CreateCustomerProductLineAccess(ctx context.Context, req *CreateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError)
	UpdateCustomerProductLineAccess(ctx context.Context, req *UpdateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError)
	DeleteCustomerProductLineAccess(ctx context.Context, req *DeleteCustomerProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type CustomerProductLineAccessSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type customerProductLineAccessSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var customerProductLineAccessSvcTracer = tracing.GetTracer("api-gateway.endpoints.customer-product-line-access.service")

func (c *CustomerProductLineAccessSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("customer product line access endpoint service: core client is required")
	}
	return nil
}

func NewCustomerProductLineAccessSvc(config *CustomerProductLineAccessSvcConfig) CustomerProductLineAccessSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &customerProductLineAccessSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *customerProductLineAccessSvcImpl) ListCustomerProductLineAccess(ctx context.Context, req *ListCustomerProductLineAccessRequest) (*apiresource.List[apiresource.CustomerProductLineAccess], *apierror.APIError) {
	pbReq := &pb.ListCustomerProductLineAccessRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerProductLineAccessSvcTracer, "service.customer_product_line_access.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCustomerProductLineAccessResponse, error) {
			return m.coreClient.ListCustomerProductLineAccess(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return CustomerProductLineAccessListPresenter(resp), nil
}

func (m *customerProductLineAccessSvcImpl) GetCustomerProductLineAccess(ctx context.Context, req *GetCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
	pbReq := &pb.GetCustomerProductLineAccessRequest{
		CustomerId: req.CustomerID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerProductLineAccessSvcTracer, "service.customer_product_line_access.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCustomerProductLineAccessResponse, error) {
			return m.coreClient.GetCustomerProductLineAccess(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CustomerProductLineAccessPresenter(resp.Item)
	return &result, nil
}

func (m *customerProductLineAccessSvcImpl) CreateCustomerProductLineAccess(ctx context.Context, req *CreateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
	pbReq := &pb.CreateCustomerProductLineAccessRequest{
		CustomerId:     req.CustomerID,
		ProductLineIds: req.ProductLineIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerProductLineAccessSvcTracer, "service.customer_product_line_access.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCustomerProductLineAccessResponse, error) {
			return m.coreClient.CreateCustomerProductLineAccess(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CustomerProductLineAccessPresenter(resp.Item)
	return &result, nil
}

func (m *customerProductLineAccessSvcImpl) UpdateCustomerProductLineAccess(ctx context.Context, req *UpdateCustomerProductLineAccessRequest) (*apiresource.CustomerProductLineAccess, *apierror.APIError) {
	pbReq := &pb.UpdateCustomerProductLineAccessRequest{
		CustomerId: req.CustomerID,
	}
	if req.ProductLineIDs != nil {
		pbReq.ProductLineIds = *req.ProductLineIDs
	}

	resp, apiErr := grpcutil.CallRPC(ctx, customerProductLineAccessSvcTracer, "service.customer_product_line_access.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateCustomerProductLineAccessResponse, error) {
			return m.coreClient.UpdateCustomerProductLineAccess(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := CustomerProductLineAccessPresenter(resp.Item)
	return &result, nil
}

func (m *customerProductLineAccessSvcImpl) DeleteCustomerProductLineAccess(ctx context.Context, req *DeleteCustomerProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteCustomerProductLineAccessRequest{
		CustomerId: req.CustomerID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, customerProductLineAccessSvcTracer, "service.customer_product_line_access.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteCustomerProductLineAccess(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
