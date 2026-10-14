package sandboxep

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

// Request to delete a sandbox.
type DeleteSandboxRequest struct {
	// ID of the sandbox to delete.
	SandboxID string `path:"id" validate:"required"`
}

// Deletes a sandbox account and everything inside it.
//
// The sandbox becomes inaccessible as soon as this call returns, but its data is purged asynchronously and may persist briefly. Deletion is permanent: the sandbox cannot be restored, and deleting it again reports that it has already been deleted. Sandboxes cannot be deleted while acting in a sandbox.
type DeleteSandboxEndpoint struct{}

func (e *DeleteSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSandboxRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteSandboxRequest, *object.Deleted]{
		Title:             "Delete Sandbox",
		Method:            http.MethodDelete,
		Route:             "/v1/core/sandboxes/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		ObjectType:        constants.ObjectTypeSandbox,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSandbox, Action: types.ActionDelete}}},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSandboxRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(SandboxSvc).DeleteSandbox, constants.ObjectTypeSandbox, "id")
		},
	})
}
