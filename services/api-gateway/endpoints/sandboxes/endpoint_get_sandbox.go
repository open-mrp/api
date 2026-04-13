package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a sandbox.
type GetSandboxRequest struct {
	// Sandbox ID.
	SandboxID string `path:"id" validate:"required"`
}

type GetSandboxEndpoint struct{}

func (e *GetSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSandboxRequest, *apiresource.Sandbox] {
	return &apiendpoint.APIEndpoint[*GetSandboxRequest, *apiresource.Sandbox]{
		Title:             "Get Sandbox",
		Description:       "Returns a sandbox by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/core/sandboxes/{id}",
		ContentType:       "application/json",
		Request:           &GetSandboxRequest{},
		Response:          &apiresource.Sandbox{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
			return svc.(SandboxSvc).GetSandbox
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSandbox,
			Fields:     []string{"owner_account"},
		}),
	}
}
