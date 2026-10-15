package locationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list locations.
type ListLocationsRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return locations with one of these statuses. Omit to return all statuses.
	Statuses []constants.ArchiveStatus `query:"statuses"`
	// Only return the locations directly under one of these locations.
	ParentIDs []string `query:"parent_ids" validate:"omitempty,max=100"`
}

// Returns a list of locations.
//
// Every location in your account is returned regardless of its depth in the hierarchy, newest first. Use `parent_ids` to list one location's children.
type ListLocationsEndpoint struct{}

func (e *ListLocationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListLocationsRequest, *apiresource.List[apiresource.Location]] {
	return (&apiendpoint.APIEndpoint[*ListLocationsRequest, *apiresource.List[apiresource.Location]]{
		Title:             "List Locations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionRead}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListLocationsRequest) (*apiresource.List[apiresource.Location], *apierror.APIError) {
			return svc.(LocationSvc).ListLocations
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"address", "parent", "children"},
		}),
		ObjectType: constants.ObjectTypeLocation,
	})
}
