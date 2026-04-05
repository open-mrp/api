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

// CreateAccountIntegrationRequest is the request to create or upsert an account integration.
type CreateAccountIntegrationRequest struct {
	// The human-readable name for the integration.
	Name string `json:"name" validate:"required"`
	// The integration provider code (e.g. "stripe", "shippo").
	IntegrationCode constants.IntegrationCode `json:"integration_code" validate:"required"`
	// The credentials JSON string containing provider-specific keys.
	Credentials string `json:"credentials" validate:"required"`
}

var sampleCreateAccountIntegrationRequest = &CreateAccountIntegrationRequest{
	Name:            "My Stripe Integration",
	IntegrationCode: constants.IntegrationCodeStripe,
	Credentials:     `{"privateKey":"sk_test_...","publishableKey":"pk_test_...","webhookSecret":"whsec_..."}`, // #nosec G101 -- sample request data
}

func (*CreateAccountIntegrationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountIntegrationRequest)
}

type CreateAccountIntegrationEndpoint struct{}

func (e *CreateAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountIntegrationRequest, *apiresource.AccountIntegration] {
	return &apiendpoint.APIEndpoint[*CreateAccountIntegrationRequest, *apiresource.AccountIntegration]{
		Title:             "Create Account Integration",
		Description:       "Creates a new account integration, or updates an existing one with the same integration code. Credentials are encrypted at rest and never returned in API responses.",
		Method:            http.MethodPost,
		Route:             "/v1/identity/integrations",
		Request:           &CreateAccountIntegrationRequest{},
		Response:          &apiresource.AccountIntegration{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		Extras: apiendpoint.APIEndpointExtras{
			ShieldRequestBody: true,
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).CreateAccountIntegration
		},
	}
}
