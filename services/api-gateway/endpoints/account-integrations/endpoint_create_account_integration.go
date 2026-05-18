package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create or upsert an account integration.
type CreateAccountIntegrationRequest struct {
	// Display name of the integration.
	Name string `json:"name" validate:"required,max=255"`
	// Integration provider code (e.g. "stripe", "shippo").
	IntegrationCode constants.IntegrationCode `json:"integration_code" validate:"required"`
	// Credentials JSON string containing provider-specific keys.
	Credentials string `json:"credentials" validate:"required" sensitive:"true"`
}

var sampleCreateAccountIntegrationRequest = &CreateAccountIntegrationRequest{
	Name:            "My Stripe Integration",
	IntegrationCode: constants.IntegrationCodeStripe,
	Credentials:     `{"privateKey":"sk_test_...","publishableKey":"pk_test_...","webhookSecret":"whsec_..."}`, // #nosec G101 -- sample request data
}

func (*CreateAccountIntegrationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountIntegrationRequest)
}

// Creates an account integration, or updates an existing one with the same integration code. Credentials are encrypted at rest and never returned in API responses.
type CreateAccountIntegrationEndpoint struct{}

func (e *CreateAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountIntegrationRequest, *apiresource.AccountIntegration] {
	return (&apiendpoint.APIEndpoint[*CreateAccountIntegrationRequest, *apiresource.AccountIntegration]{
		Title:             "Create Account Integration",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/identity/integrations",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).CreateAccountIntegration
		},
		LocationFunc: func(resp *apiresource.AccountIntegration) string {
			return "/v1/identity/integrations/" + resp.ID
		},
	})
}
