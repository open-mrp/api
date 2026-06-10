package productep

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/export"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var validateProductIncludes = []string{"product_line", "item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"}

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
	ExportProducts(ctx context.Context, req *ExportProductsRequest) (*httptransport.FileDownload, *apierror.APIError)
}

type ProductSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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

	ids := make([]string, len(resp.Products))
	for i, p := range resp.Products {
		ids[i] = p.Id
	}
	loaded, apiErr := resourceloaders.LoadProducts(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	products := make([]apiresource.Product, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			products = append(products, *(v.(*apiresource.Product)))
		}
	}
	return apiresource.NewList(products, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *productSvcImpl) GetProduct(ctx context.Context, req *RetrieveProductRequest) (*apiresource.Product, *apierror.APIError) {
	return loadProductByID(ctx, req.ProductID)
}

func (m *productSvcImpl) CreateProduct(ctx context.Context, req *CreateProductRequest) (*apiresource.Product, *apierror.APIError) {
	isPortalReady := false
	if v, ok := req.PortalVisibility.Value(); ok {
		isPortalReady = v == constants.CustomerPortalVisibilityVisible
	}

	pbReq := &pb.CreateProductRequest{
		Sku:             req.SKU,
		Description:     req.Description.Ptr(),
		Notes:           req.Notes.Ptr(),
		ProductTypeCode: string(req.ProductTypeCode),
		ProductLineId:   req.ProductLineID.Ptr(),
		CategoryId:      req.CategoryID,
		IsPortalReady:   isPortalReady,
		UnitPrice:       rateInputToProto(req.UnitPrice.Ptr()),
		UnitCost:        rateInputToProto(req.UnitCost.Ptr()),
		AttributeIds:    req.AttributeIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateProductResponse, error) {
			return m.coreClient.CreateProduct(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadProductByID(ctx, resp.Product.Id)
}

func (m *productSvcImpl) UpdateProduct(ctx context.Context, req *UpdateProductRequest) (*apiresource.Product, *apierror.APIError) {
	var isPortalReady *bool
	if pv, ok := req.PortalVisibility.Value(); ok {
		v := pv == constants.CustomerPortalVisibilityVisible
		isPortalReady = &v
	}

	pbReq := &pb.UpdateProductRequest{
		Id:            req.ProductID,
		Sku:           req.SKU.Ptr(),
		Description:   field.StringClearableToProto(req.Description),
		Notes:         field.StringClearableToProto(req.Notes),
		IsPortalReady: isPortalReady,
		UnitPrice:     rateInputToProto(req.UnitPrice.Ptr()),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateProductResponse, error) {
			return m.coreClient.UpdateProduct(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadProductByID(ctx, resp.Product.Id)
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
	stashProductMeta(ctx, resp.Product)
	return &result, nil
}

func (m *productSvcImpl) ChangeProductProductLine(ctx context.Context, req *ChangeProductProductLineRequest) (*apiresource.Product, *apierror.APIError) {
	pbReq := &pb.ChangeProductProductLineRequest{
		Id:            req.ProductID,
		ProductLineId: req.ProductLineID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.change-product-line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangeProductProductLineResponse, error) {
			return m.coreClient.ChangeProductProductLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadProductByID(ctx, resp.Product.Id)
}

func (m *productSvcImpl) ValidateProducts(ctx context.Context, req *ValidateProductsRequest) (*apiresource.ValidateProductsResponse, *apierror.APIError) {
	pbReq := &pb.ValidateProductsRequest{
		ProductsMap: req.ProductsMap,
		Includes:    resourcekit.FilterIncludes(ctx, validateProductIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.validate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ValidateProductsResponse, error) {
			return m.coreClient.ValidateProducts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	for _, proto := range resp.Products {
		meta.Set(constants.ObjectTypeProduct, proto.Id, "item_id", proto.ItemId)
		if proto.ProductLineId != nil {
			meta.Set(constants.ObjectTypeProduct, proto.Id, "product_line_id", *proto.ProductLineId)
		}
	}

	result := ValidateProductsPresenter(resp)
	for _, p := range result.Products {
		p.Item = nil
		p.ProductLine = nil
	}
	return result, nil
}

func (m *productSvcImpl) ExportProducts(ctx context.Context, req *ExportProductsRequest) (*httptransport.FileDownload, *apierror.APIError) {
	pbReq := &pb.ExportProductsRequest{
		Query:          req.Query,
		CustomerIds:    req.CustomerIDs,
		ProductLineIds: req.ProductLineIDs,
		CategoryIds:    req.CategoryIDs,
		AttributeIds:   req.AttributeIDs,
	}
	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, productSvcTracer, "service.products.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportProductsResponse, error) {
			return m.coreClient.ExportProducts(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	products := make([]apiresource.Product, len(resp.Products))
	for i, p := range resp.Products {
		products[i] = ProductPresenter(p)
	}

	body, err := export.ProductsToExcel(products)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to build export file.")
	}

	// Generate filename with current date (e.g. products_2026-05-14.xlsx)
	dateStr := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("products_%s.xlsx", dateStr)

	return &httptransport.FileDownload{
		ContentType: export.ExcelContentType,
		Filename:    filename,
		Body:        body,
	}, nil
}

func stashProductMeta(ctx context.Context, p *pb.ProductFullInfo) {
	if p == nil {
		return
	}
	meta := resourcekit.GetLoadMeta(ctx)
	meta.Set(constants.ObjectTypeProduct, p.Id, "item_id", p.ItemId)
	if p.ProductLineId != nil {
		meta.Set(constants.ObjectTypeProduct, p.Id, "product_line_id", *p.ProductLineId)
	}
}

func loadProductByID(ctx context.Context, id string) (*apiresource.Product, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadProducts(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Product not found.")
	}
	return v.(*apiresource.Product), nil
}
