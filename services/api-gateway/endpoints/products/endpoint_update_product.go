package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateProductRequest is the request to partially update a product.
type UpdateProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
	// SKU.
	SKU *string `json:"sku,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Description.
	Description *string `json:"description,omitempty" nullable:"false"`
	// Notes.
	Notes *string `json:"notes,omitempty" nullable:"false"`
	// Whether visible in the customer portal.
	PortalVisibility *constants.CustomerPortalVisibility `json:"portal_visibility,omitempty" nullable:"false"`
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
