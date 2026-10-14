package accountintegrationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list account integrations.
type ListAccountIntegrationsRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Returns a paginated list of the third-party providers connected to the target account.
//
// Stored credentials are never included in the response.
type ListAccountIntegrationsEndpoint struct{}

func (e *ListAccountIntegrationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountIntegrationsRequest, *apiresource.List[apiresource.AccountIntegration]] {
	return (&apiendpoint.APIEndpoint[*ListAccountIntegrationsRequest, *apiresource.List[apiresource.AccountIntegration]]{
		Title:             "List Account Integrations",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/settings/integrations",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{RoleType: constants.RoleTypeAdmin},
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountIntegrationsRequest) (*apiresource.List[apiresource.AccountIntegration], *apierror.APIError) {
			return svc.(AccountIntegrationSvc).ListAccountIntegrations
		},
	})
}
