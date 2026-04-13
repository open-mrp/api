package productlineep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	ownerutil "github.com/augno/api/services/api-gateway/internal/owner"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ProductLineSvc interface {
	ListProductLines(ctx context.Context, req *ListProductLinesRequest) (*apiresource.List[apiresource.ProductLine], *apierror.APIError)
	GetProductLine(ctx context.Context, req *GetProductLineRequest) (*apiresource.ProductLine, *apierror.APIError)
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

	var ownerAccount *apiresource.Account
	for _, pl := range resp.ProductLines {
		if pl.AccountId != nil {
			ownerAccount = ownerutil.ResolveOwnerAccount(ctx, m.coreClient, pl.AccountId)
			break
		}
	}

	return ProductLineListPresenter(resp, ownerAccount), nil
}

func (m *productLineSvcImpl) GetProductLine(ctx context.Context, req *GetProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
	pbReq := &pb.GetProductLineRequest{
		Id: req.ProductLineID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productLineSvcTracer, "service.product-lines.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductLineResponse, error) {
			return m.coreClient.GetProductLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.ProductLine.AccountId)
	result := ProductLinePresenter(resp.ProductLine, ownerAccount)
	return &result, nil
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

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.ProductLine.AccountId)
	result := ProductLinePresenter(resp.ProductLine, ownerAccount)
	return &result, nil
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

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.ProductLine.AccountId)
	result := ProductLinePresenter(resp.ProductLine, ownerAccount)
	return &result, nil
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
