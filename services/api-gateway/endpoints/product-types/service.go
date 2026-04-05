package producttypeep

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

type ProductTypeSvc interface {
	ListProductTypes(ctx context.Context, req *ListProductTypesRequest) (*apiresource.List[apiresource.ProductType], *apierror.APIError)
	GetProductType(ctx context.Context, req *GetProductTypeRequest) (*apiresource.ProductType, *apierror.APIError)
	CreateProductType(ctx context.Context, req *CreateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError)
	UpdateProductType(ctx context.Context, req *UpdateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError)
	DeleteProductType(ctx context.Context, req *DeleteProductTypeRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ProductTypeSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type productTypeSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var productTypeSvcTracer = tracing.GetTracer("api-gateway.endpoints.product-types.service")

func (c *ProductTypeSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("product type endpoint service: core client is required")
	}
	return nil
}

func NewProductTypeSvc(config *ProductTypeSvcConfig) ProductTypeSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productTypeSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *productTypeSvcImpl) ListProductTypes(ctx context.Context, req *ListProductTypesRequest) (*apiresource.List[apiresource.ProductType], *apierror.APIError) {
	pbReq := &pb.ListProductTypesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productTypeSvcTracer, "service.product_types.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductTypesResponse, error) {
			return m.coreClient.ListProductTypes(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ProductTypeListPresenter(resp), nil
}

func (m *productTypeSvcImpl) GetProductType(ctx context.Context, req *GetProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
	pbReq := &pb.GetProductTypeRequest{
		Identifier: req.ProductTypeID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productTypeSvcTracer, "service.product_types.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductTypeResponse, error) {
			return m.coreClient.GetProductType(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductTypePresenter(resp.ProductType)
	return &result, nil
}

func (m *productTypeSvcImpl) CreateProductType(ctx context.Context, req *CreateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
	pbReq := &pb.CreateProductTypeRequest{
		Name: req.Name,
		Code: req.Code,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productTypeSvcTracer, "service.product_types.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateProductTypeResponse, error) {
			return m.coreClient.CreateProductType(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductTypePresenter(resp.ProductType)
	return &result, nil
}

func (m *productTypeSvcImpl) UpdateProductType(ctx context.Context, req *UpdateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
	pbReq := &pb.UpdateProductTypeRequest{
		Id:   req.ProductTypeID,
		Name: req.Name,
		Code: req.Code,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productTypeSvcTracer, "service.product_types.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductTypeResponse, error) {
			return m.coreClient.UpdateProductType(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductTypePresenter(resp.ProductType)
	return &result, nil
}

func (m *productTypeSvcImpl) DeleteProductType(ctx context.Context, req *DeleteProductTypeRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteProductTypeRequest{
		Id: req.ProductTypeID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, productTypeSvcTracer, "service.product_types.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductType(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
