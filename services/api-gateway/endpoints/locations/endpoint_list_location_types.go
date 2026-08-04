package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list location types.
type ListLocationTypesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of location types.
//
// Location types are platform-defined and the same for every account, so this list is the complete set of levels you can assign when creating a location. The `q` search term matches on location type name.
type ListLocationTypesEndpoint struct{}

func (e *ListLocationTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListLocationTypesRequest, *apiresource.List[apiresource.LocationType]] {
	return (&apiendpoint.APIEndpoint[*ListLocationTypesRequest, *apiresource.List[apiresource.LocationType]]{
		Title:               "List Location Types",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/location-types",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListLocationTypesRequest) (*apiresource.List[apiresource.LocationType], *apierror.APIError) {
			return svc.(LocationSvc).ListLocationTypes
		},
	})
}
