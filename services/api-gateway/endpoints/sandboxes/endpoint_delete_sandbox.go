package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete a sandbox.
type DeleteSandboxRequest struct {
	// Sandbox ID.
	SandboxID string `path:"id" validate:"required"`
}

type DeleteSandboxEndpoint struct{}

func (e *DeleteSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSandboxRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteSandboxRequest, *apiresource.EmptyResource]{
		Title:             "Delete Sandbox",
		Description:       "Deletes a sandbox account. At least one sandbox must remain per production account.",
		Method:            http.MethodDelete,
		Route:             "/v1/core/sandboxes/{id}",
		ContentType:       "application/json",
		Request:           &DeleteSandboxRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteSandboxRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SandboxSvc).DeleteSandbox
		},
	}
}
