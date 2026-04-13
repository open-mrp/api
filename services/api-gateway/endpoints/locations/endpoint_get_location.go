package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a location.
type GetLocationRequest struct {
	// Location ID.
	LocationID string `path:"id" validate:"required"`
}

type GetLocationEndpoint struct{}

func (e *GetLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetLocationRequest, *apiresource.Location] {
	return &apiendpoint.APIEndpoint[*GetLocationRequest, *apiresource.Location]{
		Title:             "Get Location",
		Description:       "Returns a location by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/{id}",
		Request:           &GetLocationRequest{},
		Response:          &apiresource.Location{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetLocationRequest) (*apiresource.Location, *apierror.APIError) {
			return svc.(LocationSvc).GetLocation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"parent", "children"},
		}),
	}
}
