package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetSandboxRequest is the request to retrieve a single sandbox.
type GetSandboxRequest struct {
	// The ID of the sandbox to retrieve.
	SandboxID string `path:"id"`
}

const getSandboxEndpointDescription string = `This endpoint returns a single sandbox account by its ID.`

type GetSandboxEndpoint struct{}

func (e *GetSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSandboxRequest, *apiresource.Sandbox] {
	return &apiendpoint.APIEndpoint[*GetSandboxRequest, *apiresource.Sandbox]{
		Title:             "Get Sandbox",
		Description:       getSandboxEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/core/sandboxes/{id}",
		ContentType:       "application/json",
		Request:           &GetSandboxRequest{},
		Response:          apiresource.SampleSandbox,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
			return svc.(SandboxSvc).GetSandbox
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
