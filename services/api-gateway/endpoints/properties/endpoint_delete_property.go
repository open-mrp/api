package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeletePropertyRequest is the request to delete a property.
type DeletePropertyRequest struct {
	// The ID of the property to delete.
	PropertyID string `path:"id" validate:"required"`
}

type DeletePropertyEndpoint struct{}

func (e *DeletePropertyEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePropertyRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeletePropertyRequest, *apiresource.EmptyResource]{
		Title:             "Delete Property",
		Description:       "Deletes a property and all its associated attributes.",
		Method:            http.MethodDelete,
		Route:             "/v1/catalog/properties/{id}",
		ContentType:       "application/json",
		Request:           &DeletePropertyRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePropertyRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PropertySvc).DeleteProperty
		},
	}
}
