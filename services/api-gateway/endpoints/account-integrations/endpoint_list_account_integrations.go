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

// Request to list integrations.
type ListAccountIntegrationsRequest struct {
	object.ListRequest
	// Case-insensitive substring match on `name`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return integrations with one of these statuses. Omit to return all statuses.
	Statuses []constants.PauseStatus `query:"statuses"`
}

// Returns a list of your account's integrations.
//
// Credentials are never included. Integrations are sorted by creation date, newest first.
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
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountIntegrationsRequest) (*apiresource.List[apiresource.AccountIntegration], *apierror.APIError) {
			return svc.(AccountIntegrationSvc).ListAccountIntegrations
		},
	})
}
