package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateProductRequest is the request to create a product.
type CreateProductRequest struct {
	// SKU.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description.
	Description *string `json:"description"`
	// Notes.
	Notes *string `json:"notes"`
	// Product type code (e.g. sale, sample).
	ProductTypeCode string `json:"type" validate:"required,max=255"`
	// Product line ID.
	ProductLineID *string `json:"product_line_id" validate:"omitempty,max=191"`
	// Category ID.
	CategoryID string `json:"category_id" validate:"required,max=191"`
	// Whether visible on the customer portal.
	IsPortalReady bool `json:"is_portal_ready"`
	// Unit price.
	UnitPrice *string `json:"unit_price"`
}

var sampleCreateProductRequest = &CreateProductRequest{
	SKU:             apiresource.SampleItemSKU,
	ProductTypeCode: apiresource.SampleProductTypeCode,
	CategoryID:      apiresource.SampleItemCategoryID,
	IsPortalReady:   true,
}

func (*CreateProductRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductRequest)
}

type CreateProductEndpoint struct{}

func (e *CreateProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductRequest, *apiresource.Product] {
	return &apiendpoint.APIEndpoint[*CreateProductRequest, *apiresource.Product]{
		Title:             "Create Product",
		Description:       "Creates a product.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/products",
		Request:           &CreateProductRequest{},
		Response:          &apiresource.Product{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).CreateProduct
		},
		LocationFunc: func(resp *apiresource.Product) string {
			return "/v1/catalog/products/" + resp.ID
		},
	}
}
