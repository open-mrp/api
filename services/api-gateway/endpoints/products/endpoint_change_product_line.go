package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ChangeProductProductLineRequest is the request to change a product's product line.
type ChangeProductProductLineRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
	// Product line ID.
	ProductLineID string `path:"product_line_id" validate:"required"`
}

type ChangeProductProductLineEndpoint struct{}

func (e *ChangeProductProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeProductProductLineRequest, *apiresource.Product] {
	return &apiendpoint.APIEndpoint[*ChangeProductProductLineRequest, *apiresource.Product]{
		Title:             "Change Product Product Line",
		Description:       "Changes the product line assignment for a product.",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/products/{id}/product-line/{product_line_id}",
		ContentType:       "application/json",
		Request:           &ChangeProductProductLineRequest{},
		Response:          &apiresource.Product{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeProductProductLineRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).ChangeProductProductLine
		},
	}
}
