package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to create or upsert an account integration.
type CreateAccountIntegrationRequest struct {
	// Display name of the integration.
	Name string `json:"name" validate:"required,max=255"`
	// Integration provider code.
	//
	// - `stripe`: Stripe payment processing.
	// - `shippo`: Shippo shipping and label generation.
	// - `hubspot`: HubSpot CRM.
	IntegrationCode constants.IntegrationCode `json:"provider" validate:"required"`
	// JSON string containing the provider's credentials.
	//
	// Required keys depend on the provider:
	//
	// - `stripe`: `private_key` (`sk_...`), `publishable_key` (`pk_...`), and `webhook_secret` (`whsec_...`).
	// - `shippo`: `api_key` (`shippo_live_...` or `shippo_test_...`).
	// - `hubspot`: `access_token` (`pat-...`).
	//
	// For Stripe and Shippo, sandbox accounts must supply test keys and production accounts must supply live keys; credentials that do not match are rejected. HubSpot tokens make no such distinction.
	Credentials string `json:"credentials" validate:"required" sensitive:"true"`
}

var sampleCreateAccountIntegrationRequest = &CreateAccountIntegrationRequest{
	Name:            "My Stripe Integration",
	IntegrationCode: constants.IntegrationCodeStripe,
	Credentials:     `{"private_key":"sk_test_...","publishable_key":"pk_test_...","webhook_secret":"whsec_..."}`, // #nosec G101 -- sample request data
}

func (*CreateAccountIntegrationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountIntegrationRequest)
}

// Connects a third-party provider to the account, or replaces the name and credentials of the provider's existing connection.
//
// An account can have at most one integration per `provider`, so calling this again for a provider that is already connected rotates its credentials in place and returns the same integration rather than creating a second one. Credentials are checked for the provider's expected key format, encrypted at rest, and never returned in API responses.
type CreateAccountIntegrationEndpoint struct{}

func (e *CreateAccountIntegrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateAccountIntegrationRequest, *apiresource.AccountIntegration] {
	return (&apiendpoint.APIEndpoint[*CreateAccountIntegrationRequest, *apiresource.AccountIntegration]{
		Title:             "Create Account Integration",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/settings/integrations",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).CreateAccountIntegration
		},
		LocationFunc: func(resp *apiresource.AccountIntegration) string {
			return "/v1/settings/integrations/" + resp.ID
		},
	})
}
