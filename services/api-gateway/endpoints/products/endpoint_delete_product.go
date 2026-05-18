package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteProductRequest is the request to delete a product.
type DeleteProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
}

// Soft-deletes a product and returns the deleted product.
type DeleteProductEndpoint struct{}

func (e *DeleteProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*DeleteProductRequest, *apiresource.Product]{
		Title:             "Delete Product",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/products/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).DeleteProduct
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
