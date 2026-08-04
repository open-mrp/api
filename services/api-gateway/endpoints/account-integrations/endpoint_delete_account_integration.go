package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an account integration.
type DeleteAccountIntegrationRequest struct {
	// Account integration ID.
	AccountIntegrationID string `path:"id" validate:"required"`
}

// Disconnects a third-party provider from the account and returns the deleted integration.
//
// The stored credentials go with it, so any feature that relies on the provider stops working until the integration is created again. Deleting an integration that is already deleted returns an error rather than succeeding silently. To pause a provider without discarding its credentials, set the integration's status to `inactive` instead.
type DeleteAccountIntegrationEndpoint struct{}

func (e *DeleteAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountIntegrationRequest, *apiresource.AccountIntegration] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountIntegrationRequest, *apiresource.AccountIntegration]{
		Title:             "Delete Account Integration",
		Method:            http.MethodDelete,
		Route:             "/v1/settings/integrations/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).DeleteAccountIntegration
		},
	})
}
