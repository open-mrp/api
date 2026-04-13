package producttypeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a product type.
type DeleteProductTypeRequest struct {
	// Product type ID.
	ProductTypeID string `path:"id" validate:"required"`
}

type DeleteProductTypeEndpoint struct{}

func (e *DeleteProductTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteProductTypeRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteProductTypeRequest, *apiresource.EmptyResource]{
		Title:             "Delete Product Type",
		Description:       "Deletes a product type.",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/product-types/{id}",
		ContentType:       "application/json",
		Request:           &DeleteProductTypeRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteProductTypeRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ProductTypeSvc).DeleteProductType
		},
	}
}
