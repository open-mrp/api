package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list locations.
type ListLocationsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of locations for the caller's account.
type ListLocationsEndpoint struct{}

func (e *ListLocationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListLocationsRequest, *apiresource.List[apiresource.Location]] {
	return (&apiendpoint.APIEndpoint[*ListLocationsRequest, *apiresource.List[apiresource.Location]]{
		Title:             "List Locations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListLocationsRequest) (*apiresource.List[apiresource.Location], *apierror.APIError) {
			return svc.(LocationSvc).ListLocations
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"parent", "children"},
		}),
	})
}
