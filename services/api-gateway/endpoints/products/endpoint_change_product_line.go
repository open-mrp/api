package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ChangeProductProductLineRequest is the request to change a product's product line.
type ChangeProductProductLineRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
	// ID of the product line to assign to the product.
	ProductLineID string `path:"product_line_id" validate:"required"`
}

// Changes the product line assignment for a product.
type ChangeProductProductLineEndpoint struct{}

func (e *ChangeProductProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeProductProductLineRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*ChangeProductProductLineRequest, *apiresource.Product]{
		Title:             "Change Product Product Line",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/products/{id}/product-line/{product_line_id}",
		SDKMethodKey:      "change_product_line",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeProductProductLineRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).ChangeProductProductLine
		},
		ObjectType: constants.ObjectTypeProduct,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
