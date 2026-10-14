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
// Fails if the location has child locations; remove or reassign the children first.
type DeleteLocationEndpoint struct{}

func (e *DeleteLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteLocationRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteLocationRequest, *object.Deleted]{
		Title:             "Delete Location",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteLocationRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(LocationSvc).DeleteLocation, constants.ObjectTypeLocation, "id")
		},
	})
}
