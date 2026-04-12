package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateAccountIntegrationRequest is the request to update an account integration.
type UpdateAccountIntegrationRequest struct {
	// The ID of the account integration to update.
	AccountIntegrationID string `path:"id" validate:"required"`
	// The human-readable name for the integration.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Whether this integration is currently active.
	IsActive *bool `json:"is_active,omitempty" nullable:"false"`
}

var sampleUpdateAccountIntegrationRequest = &UpdateAccountIntegrationRequest{
	Name: ptrString("Updated Stripe Integration"),
}

func ptrString(s string) *string { return &s }

func (*UpdateAccountIntegrationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountIntegrationRequest)
}

type UpdateAccountIntegrationEndpoint struct{}

func (e *UpdateAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateAccountIntegrationRequest, *apiresource.AccountIntegration] {
	return &apiendpoint.APIEndpoint[*UpdateAccountIntegrationRequest, *apiresource.AccountIntegration]{
		Title:             "Update Account Integration",
		Description:       "Updates an account integration's name and active status.",
		Method:            http.MethodPut,
		Route:             "/v1/identity/integrations/{id}",
		ContentType:       "application/json",
		Request:           &UpdateAccountIntegrationRequest{},
		Response:          &apiresource.AccountIntegration{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).UpdateAccountIntegration
		},
	}
}
