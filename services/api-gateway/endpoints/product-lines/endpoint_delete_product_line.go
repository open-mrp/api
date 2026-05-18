package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a product line.
type DeleteProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
}

// Deletes an account-owned product line. Default system product lines cannot be deleted.
type DeleteProductLineEndpoint struct{}

func (e *DeleteProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductLineRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteProductLineRequest, *apiresource.EmptyResource]{
		Title:             "Delete Product Line",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/product-lines/{id}",
		ContentType:       "application/json",
		Request:           &DeleteProductLineRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductLineSvc).DeleteProductLine
		},
	}).WithDocSource(e)
}
