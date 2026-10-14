package locationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list location types.
type ListLocationTypesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of location types.
//
// Location types are platform-defined and the same for every account, so this list is the complete set of levels you can assign when creating a location. The `q` search term matches on location type name.
type ListLocationTypesEndpoint struct{}

func (e *ListLocationTypesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListLocationTypesRequest, *apiresource.List[apiresource.LocationType]] {
	return (&apiendpoint.APIEndpoint[*ListLocationTypesRequest, *apiresource.List[apiresource.LocationType]]{
		Title:             "List Location Types",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/location-types",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionRead}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListLocationTypesRequest) (*apiresource.List[apiresource.LocationType], *apierror.APIError) {
			return svc.(LocationSvc).ListLocationTypes
		},
	})
}
