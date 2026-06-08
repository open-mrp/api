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
	// Whether the integration is active.
	IsActive field.Optional[bool] `json:"is_active,omitzero"`
}

var sampleUpdateAccountIntegrationRequest = &UpdateAccountIntegrationRequest{
	Name: field.Some("Updated Stripe Integration"),
}

func (*UpdateAccountIntegrationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountIntegrationRequest)
}

// Updates an account integration's name and active status.
type UpdateAccountIntegrationEndpoint struct{}

func (e *UpdateAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountIntegrationRequest, *apiresource.AccountIntegration] {
	return (&apiendpoint.APIEndpoint[*UpdateAccountIntegrationRequest, *apiresource.AccountIntegration]{
		Title:             "Update Account Integration",
		Method:            http.MethodPut,
		Route:             "/v1/identity/integrations/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).UpdateAccountIntegration
		},
	})
}
