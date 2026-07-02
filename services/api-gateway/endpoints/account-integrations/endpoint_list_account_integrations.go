package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list account integrations.
type ListAccountIntegrationsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of account integrations for the target account.
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
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountIntegrationsRequest) (*apiresource.List[apiresource.AccountIntegration], *apierror.APIError) {
			return svc.(AccountIntegrationSvc).ListAccountIntegrations
		},
	})
}
