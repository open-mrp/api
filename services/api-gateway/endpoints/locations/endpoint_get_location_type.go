package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetLocationTypeRequest is the request to retrieve a single location type by ID or code.
type GetLocationTypeRequest struct {
	// The ID or code of the location type to retrieve.
	Identifier string `path:"id" validate:"required"`
}

type GetLocationTypeEndpoint struct{}

func (e *GetLocationTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetLocationTypeRequest, *apiresource.LocationType] {
	return &apiendpoint.APIEndpoint[*GetLocationTypeRequest, *apiresource.LocationType]{
		Title:             "Get Location Type",
		Description:       "Returns a single location type by its ID or code.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/location-types/{id}",
		Request:           &GetLocationTypeRequest{},
		Response:          &apiresource.LocationType{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetLocationTypeRequest) (*apiresource.LocationType, *apierror.APIError) {
			return svc.(LocationSvc).GetLocationType
		},
	}
}
