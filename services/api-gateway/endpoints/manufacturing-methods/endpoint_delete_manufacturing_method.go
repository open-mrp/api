package manufacturingmethodep

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

// Request to delete a manufacturing method.
type DeleteManufacturingMethodRequest struct {
	// Manufacturing method ID.
	ID string `path:"id" validate:"required"`
}

// Deletes a draft manufacturing method, with its operations and materials.
//
// Active and archived methods are kept for history and can't be deleted (409 `invalid_state_transition`).
type DeleteManufacturingMethodEndpoint struct{}

func (e *DeleteManufacturingMethodEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteManufacturingMethodRequest, *object.Deleted] {
	return &apiendpoint.APIEndpoint[*DeleteManufacturingMethodRequest, *object.Deleted]{
		Title:             "Delete Manufacturing Method",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             methodsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionDelete}}},
		ObjectType:        constants.ObjectTypeManufacturingMethod,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteManufacturingMethodRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ManufacturingMethodSvc).DeleteManufacturingMethod, constants.ObjectTypeManufacturingMethod, "id")
		},
	}
}
