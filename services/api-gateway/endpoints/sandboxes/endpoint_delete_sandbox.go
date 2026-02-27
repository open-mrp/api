package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// DeleteSandboxRequest is the request to delete a sandbox
type DeleteSandboxRequest struct {
	// The ID of the sandbox to delete.
	SandboxID string `path:"id"`
}

const deleteSandboxEndpointDescription string = `This endpoint deletes a sandbox account. At least one sandbox must remain
per production account. The sandbox and its account record are removed synchronously, and all
account-scoped data is purged asynchronously.`

type DeleteSandboxEndpoint struct{}

func (e *DeleteSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteSandboxRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*DeleteSandboxRequest, *apiresource.EmptyResource]{
		Title:             "Delete Sandbox",
		Description:       deleteSandboxEndpointDescription,
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
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
