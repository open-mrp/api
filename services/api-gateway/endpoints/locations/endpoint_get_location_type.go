package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a location type.
type GetLocationTypeRequest struct {
	// Location type ID or code.
	Identifier string `path:"id" validate:"required"`
}

type GetLocationTypeEndpoint struct{}

func (e *GetLocationTypeEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetLocationTypeRequest, *apiresource.LocationType] {
	return &apiendpoint.APIEndpoint[*GetLocationTypeRequest, *apiresource.LocationType]{
		Title:             "Get Location Type",
		Description:       "Returns a location type by ID or code.",
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
