package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a location type.
type RetrieveLocationTypeRequest struct {
	// Location ID or code.
	Identifier string `path:"id" validate:"required"`
}

// Returns a location type by ID or code.
type RetrieveLocationTypeEndpoint struct{}

func (e *RetrieveLocationTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveLocationTypeRequest, *apiresource.LocationType] {
	return (&apiendpoint.APIEndpoint[*RetrieveLocationTypeRequest, *apiresource.LocationType]{
		Title:             "Retrieve Location Type",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/location-types/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveLocationTypeRequest) (*apiresource.LocationType, *apierror.APIError) {
			return svc.(LocationSvc).GetLocationType
		},
	})
}
