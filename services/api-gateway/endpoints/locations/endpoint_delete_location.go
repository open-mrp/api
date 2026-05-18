package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a location.
type DeleteLocationRequest struct {
	// Location ID.
	LocationID string `path:"id" validate:"required"`
}

// Deletes a location. Fails if the location has child locations.
type DeleteLocationEndpoint struct{}

func (e *DeleteLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteLocationRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteLocationRequest, *apiresource.EmptyResource]{
		Title:             "Delete Location",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteLocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(LocationSvc).DeleteLocation
		},
	})
}
