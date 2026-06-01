package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a DC location.
type RetrieveDCLocationRequest struct {
	// DC location ID.
	DCLocationID string `path:"id" validate:"required"`
}

// Returns a DC location by ID.
type RetrieveDCLocationEndpoint struct{}

func (e *RetrieveDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveDCLocationRequest, *apiresource.DCLocation] {
	return (&apiendpoint.APIEndpoint[*RetrieveDCLocationRequest, *apiresource.DCLocation]{
		Title:             "Retrieve DC Location",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/dc-locations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeDCLocation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).GetDCLocation
		},
	})
}
