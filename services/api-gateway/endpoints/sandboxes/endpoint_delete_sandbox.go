package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
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

func (e *DeleteSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSandboxRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*DeleteSandboxRequest, *apiresource.EmptyResource]{
		Title:               "Delete Sandbox",
		Method:              http.MethodDelete,
		Route:               "/v1/core/sandboxes/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusAccepted,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSandbox, Action: types.ActionDelete}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSandboxRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SandboxSvc).DeleteSandbox
		},
	})
}
