package accountintegrationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete an integration.
type DeleteAccountIntegrationRequest struct {
	// Account integration ID.
	AccountIntegrationID string `path:"id" validate:"required"`
}

// Deletes an integration.
//
// Disconnects the provider and discards its credentials; features that use the provider stop until it is connected again. To pause an integration and keep its credentials, set its `status` to `disabled` instead. Deleting an integration that is already deleted returns 410.
type DeleteAccountIntegrationEndpoint struct{}

func (e *DeleteAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteAccountIntegrationRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteAccountIntegrationRequest, *object.Deleted]{
		Title:             "Delete Account Integration",
		Method:            http.MethodDelete,
		Route:             "/v1/settings/integrations/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{RoleType: constants.RoleTypeAdmin},
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteAccountIntegrationRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(AccountIntegrationSvc).DeleteAccountIntegration, constants.ObjectTypeAccountIntegration, "id")
		},
	})
}
