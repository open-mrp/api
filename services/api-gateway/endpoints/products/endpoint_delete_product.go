package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteProductRequest is the request to delete a product.
type DeleteProductRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
}

type DeleteProductEndpoint struct{}

func (e *DeleteProductEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductRequest, *apiresource.Product] {
	return &apiendpoint.APIEndpoint[*DeleteProductRequest, *apiresource.Product]{
		Title:             "Delete Product",
		Description:       "Soft-deletes a product and returns the deleted product.",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/products/{id}",
		ContentType:       "application/json",
		Request:           &DeleteProductRequest{},
		Response:          &apiresource.Product{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).DeleteProduct
		},
	}
}
