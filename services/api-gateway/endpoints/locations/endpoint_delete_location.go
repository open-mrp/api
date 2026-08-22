package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to delete a location.
type DeleteLocationRequest struct {
	// Location ID.
	LocationID string `path:"id" validate:"required"`
}

// Deletes a location.
//
// Fails if the location has child locations; remove or reassign the children first.
type DeleteLocationEndpoint struct{}

func (e *DeleteLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteLocationRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteLocationRequest, *apiresource.EmptyResource]{
		Title:               "Delete Location",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/operations/locations/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteLocationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(LocationSvc).DeleteLocation
		},
	})
}
