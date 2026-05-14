package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// RetrieveProductRequest is the request to retrieve a product.
type RetrieveProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
}

type RetrieveProductEndpoint struct{}

func (e *RetrieveProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveProductRequest, *apiresource.Product] {
	return &apiendpoint.APIEndpoint[*RetrieveProductRequest, *apiresource.Product]{
		Title:             "Retrieve Product",
		Description:       "Returns a product by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/products/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveProductRequest{},
		Response:          &apiresource.Product{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).GetProduct
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	}
}
