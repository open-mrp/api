package productlineep

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

type ProductLineSvc interface {
	ListProductLines(ctx context.Context, req *ListProductLinesRequest) (*apiresource.List[apiresource.ProductLine], *apierror.APIError)
	GetProductLine(ctx context.Context, req *RetrieveProductLineRequest) (*apiresource.ProductLine, *apierror.APIError)
	CreateProductLine(ctx context.Context, req *CreateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError)
	UpdateProductLine(ctx context.Context, req *UpdateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError)
	DeleteProductLine(ctx context.Context, req *DeleteProductLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ProductLineSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type productLineSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var productLineSvcTracer = tracing.GetTracer("api-gateway.endpoints.product-lines.service")

func (c *ProductLineSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("product line endpoint service: core client is required")
	}
	return nil
}

func NewProductLineSvc(config *ProductLineSvcConfig) ProductLineSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productLineSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *productLineSvcImpl) ListProductLines(ctx context.Context, req *ListProductLinesRequest) (*apiresource.List[apiresource.ProductLine], *apierror.APIError) {
	pbReq := &pb.ListProductLinesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductLinesResponse, error) {
			return m.coreClient.ListProductLines(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.ProductLines))
	for i, pl := range resp.ProductLines {
		ids[i] = pl.Id
	}
	loaded, apiErr := resourceloaders.LoadProductLines(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.ProductLine, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.ProductLine)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *productLineSvcImpl) GetProductLine(ctx context.Context, req *RetrieveProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
	return loadProductLineByID(ctx, req.ProductLineID)
}

func (m *productLineSvcImpl) CreateProductLine(ctx context.Context, req *CreateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
	pbReq := &pb.CreateProductLineRequest{
		Name:             req.Name,
		UnitGroupId:      req.UnitGroupID,
		CommissionPolicy: string(req.CommissionPolicy),
		FreightPolicy:    string(req.FreightPolicy),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateProductLineResponse, error) {
			return m.coreClient.CreateProductLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadProductLineByID(ctx, resp.ProductLine.Id)
}

func (m *productLineSvcImpl) UpdateProductLine(ctx context.Context, req *UpdateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
	pbReq := &pb.UpdateProductLineRequest{
		Id:          req.ProductLineID,
		Name:        req.Name,
		UnitGroupId: req.UnitGroupID,
	}
	if req.CommissionPolicy != nil {
		s := string(*req.CommissionPolicy)
		pbReq.CommissionPolicy = &s
	}
	if req.FreightPolicy != nil {
		s := string(*req.FreightPolicy)
		pbReq.FreightPolicy = &s
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductLineResponse, error) {
			return m.coreClient.UpdateProductLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return loadProductLineByID(ctx, resp.ProductLine.Id)
}

func (m *productLineSvcImpl) DeleteProductLine(ctx context.Context, req *DeleteProductLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteProductLineRequest{
		Id: req.ProductLineID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteProductLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

// loadProductLineByID wraps the single-ID load pattern used after every
// mutation and for the retrieve endpoint.
func loadProductLineByID(ctx context.Context, id string) (*apiresource.ProductLine, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadProductLines(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Product line not found.")
	}
	return v.(*apiresource.ProductLine), nil
}
