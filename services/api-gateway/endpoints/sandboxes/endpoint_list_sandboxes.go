package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Returns a paginated list of sandboxes.
type ListSandboxesEndpoint struct{}

func (e *ListSandboxesEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.Sandbox]] {
	return (&apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.Sandbox]]{
		Title:               "List Sandboxes",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/core/sandboxes",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSandbox, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeSandbox,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSandbox,
			Fields:     []string{"owner_account"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.Sandbox], *apierror.APIError) {
			return svc.(SandboxSvc).ListSandboxes
		},
	})
}
