package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list locations.
type ListLocationsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of locations in your account, newest first.
//
// Every location is returned regardless of its depth in the hierarchy, so top-level locations and their descendants appear side by side. The `q` search term matches on location name.
type ListLocationsEndpoint struct{}

func (e *ListLocationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListLocationsRequest, *apiresource.List[apiresource.Location]] {
	return (&apiendpoint.APIEndpoint[*ListLocationsRequest, *apiresource.List[apiresource.Location]]{
		Title:               "List Locations",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/locations",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListLocationsRequest) (*apiresource.List[apiresource.Location], *apierror.APIError) {
			return svc.(LocationSvc).ListLocations
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"parent", "children"},
		}),
		ObjectType: constants.ObjectTypeLocation,
	})
}
