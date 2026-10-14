package sandboxep

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

// Returns a paginated list of the sandboxes owned by your production account, newest first.
//
// The `q` search term matches the sandbox name. Sandboxes cannot be listed while acting in a sandbox.
type ListSandboxesEndpoint struct{}

func (e *ListSandboxesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSandboxesRequest, *apiresource.List[apiresource.Sandbox]] {
	return (&apiendpoint.APIEndpoint[*ListSandboxesRequest, *apiresource.List[apiresource.Sandbox]]{
		Title:             "List Sandboxes",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/core/sandboxes",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSandbox, Action: types.ActionRead}}},
		Preview:           true,
		ObjectType:        constants.ObjectTypeSandbox,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSandbox,
			Fields:     []string{"owner_account"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSandboxesRequest) (*apiresource.List[apiresource.Sandbox], *apierror.APIError) {
			return svc.(SandboxSvc).ListSandboxes
		},
	})
}

// ListSandboxesRequest pages through the account's sandboxes.
type ListSandboxesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}
