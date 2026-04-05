package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetLocationRequest is the request to retrieve a single location.
type GetLocationRequest struct {
	// The ID of the location to retrieve.
	LocationID string `path:"id" validate:"required"`
}

type GetLocationEndpoint struct{}

func (e *GetLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetLocationRequest, *apiresource.Location] {
	return &apiendpoint.APIEndpoint[*GetLocationRequest, *apiresource.Location]{
		Title:             "Get Location",
		Description:       "Returns a single location by its ID.",
		Method:            http.MethodGet,
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
