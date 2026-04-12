package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteDCLocationRequest is the request to delete a DC location.
type DeleteDCLocationRequest struct {
	// The ID of the DC location to delete.
	DCLocationID string `path:"id" validate:"required"`
}

type DeleteDCLocationEndpoint struct{}

func (e *DeleteDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteDCLocationRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteDCLocationRequest, *apiresource.EmptyResource]{
		Title:             "Delete DC Location",
		Description:       "Deletes a DC location.",
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
	}
}
