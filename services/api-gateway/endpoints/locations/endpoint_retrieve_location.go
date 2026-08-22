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

// Request to get a location.
type RetrieveLocationRequest struct {
	// Location ID.
	LocationID string `path:"id" validate:"required"`
}

// Returns a location by ID.
type RetrieveLocationEndpoint struct{}

func (e *RetrieveLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveLocationRequest, *apiresource.Location] {
	return (&apiendpoint.APIEndpoint[*RetrieveLocationRequest, *apiresource.Location]{
		Title:               "Retrieve Location",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/operations/locations/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionRead}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveLocationRequest) (*apiresource.Location, *apierror.APIError) {
			return svc.(LocationSvc).GetLocation
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeLocation,
			Fields:     []string{"parent", "children"},
		}),
		ObjectType: constants.ObjectTypeLocation,
	})
}
