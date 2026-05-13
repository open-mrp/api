package productep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func rateInputToProto(r *apirequest.RateInput) *pb.CreateRateInput {
	if r == nil {
		return nil
	}
	return &pb.CreateRateInput{
		Value:             r.Value,
		NumeratorUnitId:   r.NumeratorUnitID,
		DenominatorUnitId: r.DenominatorUnitID,
	}
}

type ProductSvc interface {
	ListProducts(ctx context.Context, req *ListProductsRequest) (*apiresource.List[apiresource.Product], *apierror.APIError)
	GetProduct(ctx context.Context, req *RetrieveProductRequest) (*apiresource.Product, *apierror.APIError)
	CreateProduct(ctx context.Context, req *CreateProductRequest) (*apiresource.Product, *apierror.APIError)
	UpdateProduct(ctx context.Context, req *UpdateProductRequest) (*apiresource.Product, *apierror.APIError)
	DeleteProduct(ctx context.Context, req *DeleteProductRequest) (*apiresource.Product, *apierror.APIError)
	ChangeProductProductLine(ctx context.Context, req *ChangeProductProductLineRequest) (*apiresource.Product, *apierror.APIError)
	ValidateProducts(ctx context.Context, req *ValidateProductsRequest) (*apiresource.ValidateProductsResponse, *apierror.APIError)
}

type ProductSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type productSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var productSvcTracer = tracing.GetTracer("api-gateway.endpoints.products.service")

func portalVisibilityToReadyPtr(v *constants.CustomerPortalVisibility) *bool {
	if v == nil {
		return nil
	}
	ready := *v == constants.CustomerPortalVisibilityVisible
	return &ready
}

func (c *ProductSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("product endpoint service: core client is required")
	}
	return nil
}

func NewProductSvc(config *ProductSvcConfig) ProductSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &productSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *productSvcImpl) ListProducts(ctx context.Context, req *ListProductsRequest) (*apiresource.List[apiresource.Product], *apierror.APIError) {
	pbReq := &pb.ListProductsFullRequest{
		Cursor:         req.Cursor,
		Limit:          req.Limit,
		Query:          req.Query,
		CustomerIds:    req.CustomerIDs,
		ProductLineIds: req.ProductLineIDs,
		CategoryIds:    req.CategoryIDs,
		AttributeIds:   req.AttributeIDs,
		IsPortalReady:  portalVisibilityToReadyPtr(req.PortalVisibility),
		Includes:       appctx.GetRequestedIncludeKeys(ctx),
	}
	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListProductsFullResponse, error) {
			return m.coreClient.ListProductsFull(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ProductListPresenter(resp), nil
}

func (m *productSvcImpl) GetProduct(ctx context.Context, req *RetrieveProductRequest) (*apiresource.Product, *apierror.APIError) {
	pbReq := &pb.GetProductRequest{
		Id:       req.ProductID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProductResponse, error) {
			return m.coreClient.GetProduct(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductPresenter(resp.Product)
	return &result, nil
}

func (m *productSvcImpl) CreateProduct(ctx context.Context, req *CreateProductRequest) (*apiresource.Product, *apierror.APIError) {
	isPortalReady := false
	if req.PortalVisibility != nil {
		isPortalReady = *req.PortalVisibility == constants.CustomerPortalVisibilityVisible
	}

	pbReq := &pb.CreateProductRequest{
		Sku:             req.SKU,
		Description:     req.Description,
		Notes:           req.Notes,
		ProductTypeCode: string(req.ProductTypeCode),
		ProductLineId:   req.ProductLineID,
		CategoryId:      req.CategoryID,
		IsPortalReady:   isPortalReady,
		UnitPrice:       rateInputToProto(req.UnitPrice),
		UnitCost:        rateInputToProto(req.UnitCost),
		BurnRate:        rateInputToProto(req.BurnRate),
		AttributeIds:    req.AttributeIDs,
		Includes:        appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateProductResponse, error) {
			return m.coreClient.CreateProduct(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductPresenter(resp.Product)
	return &result, nil
}

func (m *productSvcImpl) UpdateProduct(ctx context.Context, req *UpdateProductRequest) (*apiresource.Product, *apierror.APIError) {
	var isPortalReady *bool
	if req.PortalVisibility != nil {
		v := *req.PortalVisibility == constants.CustomerPortalVisibilityVisible
		isPortalReady = &v
	}

	// ApplyExplicitNulls converts an explicit JSON null for a nullable:"true"
	// *string field into &"" (pointer to empty string). The core service uses
	// UpdateDescription/UpdateNotes=true with a nil pointer to clear the column,
	// so we translate &"" → nil here before forwarding.
	description := req.Description
	updateDescription := description != nil
	if description != nil && *description == "" {
		description = nil
	}

	notes := req.Notes
	updateNotes := notes != nil
	if notes != nil && *notes == "" {
		notes = nil
	}

	pbReq := &pb.UpdateProductRequest{
		Id:                req.ProductID,
		Sku:               req.SKU,
		Description:       description,
		UpdateDescription: updateDescription,
		Notes:             notes,
		UpdateNotes:       updateNotes,
		IsPortalReady:     isPortalReady,
		UnitPrice:         rateInputToProto(req.UnitPrice),
		Includes:          appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductResponse, error) {
			return m.coreClient.UpdateProduct(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductPresenter(resp.Product)
	return &result, nil
}

func (m *productSvcImpl) DeleteProduct(ctx context.Context, req *DeleteProductRequest) (*apiresource.Product, *apierror.APIError) {
	pbReq := &pb.DeleteProductRequest{
		Id: req.ProductID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteProductResponse, error) {
			return m.coreClient.DeleteProduct(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductPresenter(resp.Product)
	return &result, nil
}

func (m *productSvcImpl) ChangeProductProductLine(ctx context.Context, req *ChangeProductProductLineRequest) (*apiresource.Product, *apierror.APIError) {
	pbReq := &pb.ChangeProductProductLineRequest{
		Id:            req.ProductID,
		ProductLineId: req.ProductLineID,
		Includes:      appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.change-product-line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangeProductProductLineResponse, error) {
			return m.coreClient.ChangeProductProductLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ProductPresenter(resp.Product)
	return &result, nil
}

func (m *productSvcImpl) ValidateProducts(ctx context.Context, req *ValidateProductsRequest) (*apiresource.ValidateProductsResponse, *apierror.APIError) {
	pbReq := &pb.ValidateProductsRequest{
		ProductsMap: req.ProductsMap,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.validate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ValidateProductsResponse, error) {
			return m.coreClient.ValidateProducts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ValidateProductsPresenter(resp), nil
}
