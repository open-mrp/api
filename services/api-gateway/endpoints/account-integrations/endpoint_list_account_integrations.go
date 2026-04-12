package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListAccountIntegrationsRequest is the request to list account integrations.
type ListAccountIntegrationsRequest struct {
	apiresource.PaginationRequest
}

type ListAccountIntegrationsEndpoint struct{}

func (e *ListAccountIntegrationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAccountIntegrationsRequest, *apiresource.List[apiresource.AccountIntegration]] {
	return &apiendpoint.APIEndpoint[*ListAccountIntegrationsRequest, *apiresource.List[apiresource.AccountIntegration]]{
		Title:             "List Account Integrations",
		Description:       "Returns a paginated list of account integrations for the target account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/identity/integrations",
		Request:           &ListAccountIntegrationsRequest{},
		Response:          &apiresource.List[apiresource.AccountIntegration]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAccountIntegrationsRequest) (*apiresource.List[apiresource.AccountIntegration], *apierror.APIError) {
			return svc.(AccountIntegrationSvc).ListAccountIntegrations
		},
	}
}
