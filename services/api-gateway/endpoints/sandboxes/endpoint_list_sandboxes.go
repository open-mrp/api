package sandboxep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

const listSandboxesEndpointDescription string = `This endpoint returns a paginated list of sandbox accounts for the target account.
Supports cursor-based pagination.`

type ListSandboxesEndpoint struct{}

func (e *ListSandboxesEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.Sandbox]] {
	return &apiendpoint.APIEndpoint[*apiresource.PaginationRequest, *apiresource.List[apiresource.Sandbox]]{
		Title:             "List Sandboxes",
		Description:       listSandboxesEndpointDescription,
		Method:            http.MethodGet,
		Route:             "/v1/core/sandboxes",
		Request:           &apiresource.PaginationRequest{},
		Response:          &apiresource.List[apiresource.Sandbox]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: &apiendpoint.IncludeConfig{
			Fields: []apiendpoint.IncludeField{
				{Key: "owner_account", ObjectType: constants.ObjectTypeAccount, JSONPaths: []string{"owner_account"}},
			},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.Sandbox], *apierror.APIError) {
			return svc.(SandboxSvc).ListSandboxes
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
