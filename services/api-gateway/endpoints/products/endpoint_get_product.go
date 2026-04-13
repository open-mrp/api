package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetProductRequest is the request to retrieve a product.
type GetProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
}

type GetProductEndpoint struct{}

func (e *GetProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductRequest, *apiresource.Product] {
	return &apiendpoint.APIEndpoint[*GetProductRequest, *apiresource.Product]{
		Title:             "Get Product",
		Description:       "Returns a product by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/products/{id}",
		ContentType:       "application/json",
		Request:           &GetProductRequest{},
		Response:          &apiresource.Product{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).GetProduct
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_type", "product_line", "product_line.unit_group", "item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	}
}
