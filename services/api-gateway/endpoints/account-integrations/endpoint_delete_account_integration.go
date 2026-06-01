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

// Deletes an account integration and returns the deleted resource.
type DeleteAccountIntegrationEndpoint struct{}

func (e *DeleteAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountIntegrationRequest, *apiresource.AccountIntegration] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountIntegrationRequest, *apiresource.AccountIntegration]{
		Title:             "Delete Account Integration",
		Method:            http.MethodDelete,
		Route:             "/v1/identity/integrations/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).DeleteAccountIntegration
		},
	})
}
