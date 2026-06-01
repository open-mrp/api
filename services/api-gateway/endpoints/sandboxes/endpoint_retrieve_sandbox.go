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
type RetrieveSandboxRequest struct {
	// Sandbox ID.
	SandboxID string `path:"id" validate:"required"`
}

// Returns a sandbox by ID.
type RetrieveSandboxEndpoint struct{}

func (e *RetrieveSandboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSandboxRequest, *apiresource.Sandbox] {
	return (&apiendpoint.APIEndpoint[*RetrieveSandboxRequest, *apiresource.Sandbox]{
		Title:             "Retrieve Sandbox",
		Method:            http.MethodGet,
		Route:             "/v1/core/sandboxes/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSandbox,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSandboxRequest) (*apiresource.Sandbox, *apierror.APIError) {
			return svc.(SandboxSvc).GetSandbox
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSandbox,
			Fields:     []string{"owner_account"},
		}),
	})
}
