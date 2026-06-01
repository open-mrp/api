package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
)

// UpdateProductRequest is the request to partially update a product.
type UpdateProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
	// SKU.
	SKU *string `json:"sku,omitempty" validate:"omitempty,max=255"`
	// Description.
	Description *patch.Field[string] `json:"description"`
	// Notes.
	Notes *patch.Field[string] `json:"notes"`
	// Whether visible in the customer portal.
	PortalVisibility *constants.CustomerPortalVisibility `json:"portal_visibility,omitempty"`
	// Updated unit price. Numerator must be a currency unit; denominator must not be.
	UnitPrice patch.Nullable[apirequest.RateInput] `json:"unit_price,omitzero"`
}

var sampleUpdateProductSKU = "SKU-002"

var sampleUpdateProductRequest = &UpdateProductRequest{
	SKU: &sampleUpdateProductSKU,
}

func (*UpdateProductRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductRequest)
}

// Partially updates a product.
type UpdateProductEndpoint struct{}

func (e *UpdateProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*UpdateProductRequest, *apiresource.Product]{
		Title:             "Update Product",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/products/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).UpdateProduct
		},
		ObjectType: constants.ObjectTypeProduct,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
