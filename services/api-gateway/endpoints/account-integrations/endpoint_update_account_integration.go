package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update an account integration.
type UpdateAccountIntegrationRequest struct {
	// Account integration ID.
	AccountIntegrationID string `path:"id" validate:"required"`
	// Display name of the integration.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Lifecycle status of the integration.
	//
	// Set to `inactive` to stop the provider being used while keeping its stored credentials, and back to `active` to resume without re-entering them.
	Status field.Optional[constants.AccountIntegrationStatus] `json:"status,omitzero"`
}

var sampleUpdateAccountIntegrationRequest = &UpdateAccountIntegrationRequest{
	Name:   field.Some("Updated Stripe Integration"),
	Status: field.Some(constants.AccountIntegrationStatusActive),
}

func (*UpdateAccountIntegrationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountIntegrationRequest)
}

// Renames an account integration, or activates or deactivates it.
//
// Omitted fields are left unchanged. Credentials cannot be changed here; to rotate them, call Create Account Integration again with the same `provider`.
type UpdateAccountIntegrationEndpoint struct{}

func (e *UpdateAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountIntegrationRequest, *apiresource.AccountIntegration] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountIntegrationRequest, *apiresource.AccountIntegration]{
		Title:             "Update Account Integration",
		Method:            http.MethodPut,
		Route:             "/v1/settings/integrations/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).UpdateAccountIntegration
		},
	})
}
