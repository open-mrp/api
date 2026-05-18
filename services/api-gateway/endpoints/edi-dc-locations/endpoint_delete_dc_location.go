package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a DC location.
type DeleteDCLocationRequest struct {
	// DC location ID.
	DCLocationID string `path:"id" validate:"required"`
}

// Deletes a DC location.
type DeleteDCLocationEndpoint struct{}

func (e *DeleteDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteDCLocationRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteDCLocationRequest, *apiresource.EmptyResource]{
		Title:             "Delete DC Location",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/dc-locations/{id}",
		Request:           &DeleteDCLocationRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteDCLocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).DeleteDCLocation
		},
	}).WithDocSource(e)
}
