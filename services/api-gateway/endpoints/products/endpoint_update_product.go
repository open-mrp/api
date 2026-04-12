package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateProductRequest is the request to partially update a product.
type UpdateProductRequest struct {
	// The ID of the product to update.
	ProductID string `path:"id" validate:"required"`
	// The stock keeping unit code.
	SKU *string `json:"sku,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// A description of the product.
	Description *string `json:"description,omitempty" nullable:"false"`
	// Additional notes about the product.
	Notes *string `json:"notes,omitempty" nullable:"false"`
	// Whether this product is visible on the customer portal.
	IsPortalReady *bool `json:"is_portal_ready,omitempty" nullable:"false"`
}

var sampleUpdateProductSKU = "SKU-002"

var sampleUpdateProductRequest = &UpdateProductRequest{
	SKU: &sampleUpdateProductSKU,
}

func (*UpdateProductRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductRequest)
}

type UpdateProductEndpoint struct{}

func (e *UpdateProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductRequest, *apiresource.Product] {
	return &apiendpoint.APIEndpoint[*UpdateProductRequest, *apiresource.Product]{
		Title:             "Update Product",
		Description:       "Partially updates a product.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/products/{id}",
		ContentType:       "application/json",
		Request:           &UpdateProductRequest{},
		Response:          &apiresource.Product{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).UpdateProduct
		},
	}
}
