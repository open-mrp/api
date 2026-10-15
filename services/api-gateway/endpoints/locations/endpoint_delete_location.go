package locationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete a location.
type DeleteLocationRequest struct {
	// Location ID.
	LocationID string `path:"id" validate:"required"`
}

// Deletes a location.
//
// Only while no stock, inventory movement, delivery, work center or batch uses it, it has no child locations, and it isn't your default site. Archive a location instead by setting `status` to `archived`.
type DeleteLocationEndpoint struct{}

func (e *DeleteLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteLocationRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteLocationRequest, *object.Deleted]{
		Title:             "Delete Location",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeLocation,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteLocationRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(LocationSvc).DeleteLocation, constants.ObjectTypeLocation, "id")
		},
	})
}
