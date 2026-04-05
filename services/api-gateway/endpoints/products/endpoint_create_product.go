package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateProductRequest is the request to create a new product.
type CreateProductRequest struct {
	// The stock keeping unit code for the product.
	SKU string `json:"sku" validate:"required"`
	// A description of the product.
	Description *string `json:"description"`
	// Additional notes about the product.
	Notes *string `json:"notes"`
	// The product type code (e.g. sale, sample).
	ProductTypeCode string `json:"product_type_code" validate:"required"`
	// The ID of the product line to assign to this product.
	ProductLineID *string `json:"product_line_id"`
	// The ID of the item category.
	CategoryID string `json:"category_id" validate:"required"`
	// Whether this product is visible on the customer portal.
	IsPortalReady bool `json:"is_portal_ready"`
	// The unit price for this product.
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
		Description:       "Creates a new product.",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/products",
		Request:           &CreateProductRequest{},
		Response:          &apiresource.Product{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).CreateProduct
		},
	}
}
