package accountintegrationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to connect a provider.
type CreateAccountIntegrationRequest struct {
	// Display name of the integration.
	Name string `json:"name" validate:"required,max=255"`
	// The provider to connect.
	//
	// - `stripe`: Stripe payment processing.
	// - `shippo`: Shippo shipping rates and labels.
	// - `hubspot`: HubSpot CRM.
	IntegrationCode constants.IntegrationCode `json:"provider" validate:"required"`
	// The provider's credentials. Which keys are required depends on `provider`.
	//
	// For Stripe and Shippo, sandbox accounts must use test keys and live accounts live keys.
	Credentials *IntegrationCredentials `json:"credentials" validate:"required" sensitive:"true"`
	// Whether OpenMRP uses this integration: `active` or `disabled`. Defaults to `active`.
	Status field.Optional[constants.PauseStatus] `json:"status,omitzero"`
}

var sampleCreateAccountIntegrationRequest = &CreateAccountIntegrationRequest{
	Name:            "My Stripe Integration",
	IntegrationCode: constants.IntegrationCodeStripe,
	Credentials: &IntegrationCredentials{
		PrivateKey:     field.Some("sk_test_..."),
		PublishableKey: field.Some("pk_test_..."),
		WebhookSecret:  field.Some("whsec_..."), // #nosec G101 -- sample request data
	},
}

func (*CreateAccountIntegrationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateAccountIntegrationRequest)
}

// Connects a provider to your account.
//
// One connection per provider. `credentials` depends on `provider` and is never returned. Sandbox accounts must use test keys, and live accounts live keys.
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
		Auth:              authz.Policy{RoleType: constants.RoleTypeAdmin},
		ObjectType:        constants.ObjectTypeAccountIntegration,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateAccountIntegrationRequest) (*apiresource.AccountIntegration, *apierror.APIError) {
			return svc.(AccountIntegrationSvc).CreateAccountIntegration
		},
		LocationFunc: func(resp *apiresource.AccountIntegration) string {
			return "/v1/settings/integrations/" + resp.ID
		},
	})
}
