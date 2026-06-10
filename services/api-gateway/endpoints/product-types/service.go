package producttypeep

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
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductTypeSvc interface {
	ListProductTypes(ctx context.Context, req *ListProductTypesRequest) (*apiresource.List[apiresource.ProductType], *apierror.APIError)
	GetProductType(ctx context.Context, req *RetrieveProductTypeRequest) (*apiresource.ProductType, *apierror.APIError)
	CreateProductType(ctx context.Context, req *CreateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError)
	UpdateProductType(ctx context.Context, req *UpdateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError)
	DeleteProductType(ctx context.Context, req *DeleteProductTypeRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ProductTypeSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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
	return &productTypeSvcImpl{coreClient: config.CoreClient}
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
	ids := make([]string, len(resp.ProductTypes))
	for i, pt := range resp.ProductTypes {
		ids[i] = pt.Id
	}
	loaded, apiErr := resourceloaders.LoadProductTypes(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.ProductType, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.ProductType)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *productTypeSvcImpl) GetProductType(ctx context.Context, req *RetrieveProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
	// GetProductType accepts an ID or a code path param. Call the legacy
	// id-or-code lookup first, then fan back through the loader so the result
	// shape matches every other V2-migrated resource.
	resp, apiErr := grpcutil.CallRPC(ctx, productTypeSvcTracer, "service.product_types.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductTypeResponse, error) {
			return m.coreClient.GetProductType(ctx, &pb.GetProductTypeRequest{Identifier: req.ProductTypeID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadProductTypeByID(ctx, resp.ProductType.Id)
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
	return loadProductTypeByID(ctx, resp.ProductType.Id)
}

func (m *productTypeSvcImpl) UpdateProductType(ctx context.Context, req *UpdateProductTypeRequest) (*apiresource.ProductType, *apierror.APIError) {
	pbReq := &pb.UpdateProductTypeRequest{
		Id:   req.ProductTypeID,
		Name: req.Name.Ptr(),
		Code: req.Code.Ptr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, productTypeSvcTracer, "service.product_types.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductTypeResponse, error) {
			return m.coreClient.UpdateProductType(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadProductTypeByID(ctx, resp.ProductType.Id)
}

func (m *productTypeSvcImpl) DeleteProductType(ctx context.Context, req *DeleteProductTypeRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteProductTypeRequest{Id: req.ProductTypeID}
	_, apiErr := grpcutil.CallRPC(ctx, productTypeSvcTracer, "service.product_types.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductType(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// loadProductTypeByID wraps the single-ID load pattern used after every
// mutation and (chained after the id-or-code lookup) for the retrieve endpoint.
func loadProductTypeByID(ctx context.Context, id string) (*apiresource.ProductType, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadProductTypes(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Product type not found.")
	}
	return v.(*apiresource.ProductType), nil
}
