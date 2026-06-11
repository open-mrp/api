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
	// Integration provider code.
	//
	// - `stripe`: Stripe payment processing.
	// - `shippo`: Shippo shipping and label generation.
	IntegrationCode constants.IntegrationCode `json:"integration_code" validate:"required"`
	// JSON string containing the provider's credentials.
	//
	// Required keys depend on the provider:
	//
	// - `stripe`: `privateKey` (`sk_...`), `publishableKey` (`pk_...`), and `webhookSecret` (`whsec_...`).
	// - `shippo`: `apiKey` (`shippo_live_...` or `shippo_test_...`).
	//
	// Sandbox accounts must use test keys and production accounts must use live keys; credentials that do not match are rejected.
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

// Creates an account integration, or updates the name and credentials of an existing one with the same integration code.
//
// Credentials are validated for the provider, encrypted at rest, and never returned in API responses. An account can have at most one integration per integration code.
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
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).CreateAccountIntegration
		},
		LocationFunc: func(resp *apiresource.AccountIntegration) string {
			return "/v1/identity/integrations/" + resp.ID
		},
	})
}
