package catalogep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type CatalogSvc interface {
	ListCatalogProductLines(ctx context.Context, req *ListCatalogProductLinesRequest) (*apiresource.List[apiresource.CatalogProductLine], *apierror.APIError)
	ListCatalogProducts(ctx context.Context, req *ListCatalogProductsRequest) (*apiresource.List[apiresource.CatalogCategory], *apierror.APIError)
}

type CatalogSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type catalogSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var catalogSvcTracer = tracing.GetTracer("api-gateway.endpoints.catalog.service")

func (c *CatalogSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("catalog endpoint service: core client is required")
	}
	return nil
}

func NewCatalogSvc(config *CatalogSvcConfig) CatalogSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &catalogSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *catalogSvcImpl) ListCatalogProductLines(ctx context.Context, req *ListCatalogProductLinesRequest) (*apiresource.List[apiresource.CatalogProductLine], *apierror.APIError) {
	pbReq := &pb.ListCatalogProductLinesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, catalogSvcTracer, "service.catalog.list_product_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCatalogProductLinesResponse, error) {
			return m.coreClient.ListCatalogProductLines(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return CatalogProductLineListPresenter(resp), nil
}

func (m *catalogSvcImpl) ListCatalogProducts(ctx context.Context, req *ListCatalogProductsRequest) (*apiresource.List[apiresource.CatalogCategory], *apierror.APIError) {
	pbReq := &pb.ListCatalogProductsRequest{
		ProductLineId: req.ProductLineID,
		Cursor:        req.Cursor,
		Limit:         req.Limit,
		Query:         req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, catalogSvcTracer, "service.catalog.list_products", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCatalogProductsResponse, error) {
			return m.coreClient.ListCatalogProducts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	categories := make([]apiresource.CatalogCategory, len(resp.Categories))
	for i, cat := range resp.Categories {
		products := make([]apiresource.CatalogProduct, len(cat.Products))
		for j, p := range cat.Products {
			attrs := make([]apiresource.CatalogAttribute, len(p.Attributes))
			for k, a := range p.Attributes {
				attrs[k] = apiresource.CatalogAttribute{
					ID:     a.Id,
					Object: constants.ObjectTypeCatalogAttribute,
					Name:   a.Name,
					Property: &apiresource.CatalogProperty{
						ID:     a.PropertyId,
						Object: constants.ObjectTypeCatalogProperty,
						Name:   a.PropertyName,
					},
				}
			}
			products[j] = apiresource.CatalogProduct{
				Object: constants.ObjectTypeCatalogProduct,
				Item: &apiresource.Item{
					ID:     p.ItemId,
					Object: constants.ObjectTypeItem,
					SKU:    p.Sku,
				},
				Description: p.Description,
				Attributes:  attrs,
			}
		}

		properties := make([]apiresource.CatalogProperty, len(cat.Properties))
		for j, pr := range cat.Properties {
			properties[j] = apiresource.CatalogProperty{
				ID:     pr.Id,
				Object: constants.ObjectTypeCatalogProperty,
				Name:   pr.Name,
			}
		}

		categories[i] = apiresource.CatalogCategory{
			ID:         cat.Id,
			Object:     constants.ObjectTypeCatalogCategory,
			Name:       cat.Name,
			Properties: properties,
			Products:   products,
		}
	}

	return apiresource.NewList(categories, grpcutil.MapProtoPageInfo(resp.PageInfo)), nil
}
