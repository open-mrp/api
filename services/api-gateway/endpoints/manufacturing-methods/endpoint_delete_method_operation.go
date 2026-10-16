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

// Request to delete an operation.
type DeleteMethodOperationRequest struct {
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
	// Operation ID.
	ID string `path:"id" validate:"required"`
}

// Deletes an operation of a draft manufacturing method.
//
// Fails with 409 `resource_in_use` while materials are consumed at the operation; move them to another operation or delete them first. Only a `draft` method can change (409 `invalid_state_transition`).
type DeleteMethodOperationEndpoint struct{}

func (e *DeleteMethodOperationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteMethodOperationRequest, *object.Deleted] {
	return &apiendpoint.APIEndpoint[*DeleteMethodOperationRequest, *object.Deleted]{
		Title:             "Delete Method Operation",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             operationsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeMethodOperation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteMethodOperationRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(ManufacturingMethodSvc).DeleteMethodOperation, constants.ObjectTypeMethodOperation, "id")
		},
	}
}
