package accountintegrationep

import (
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// Request to replace an integration's credentials.
type RotateAccountIntegrationCredentialsRequest struct {
	lifecycle.Request
	// The provider's new credentials, with the same keys the integration's `provider` takes on create.
	Credentials *IntegrationCredentials `json:"credentials" validate:"required" sensitive:"true"`
}

var sampleRotateAccountIntegrationCredentialsRequest = &RotateAccountIntegrationCredentialsRequest{
	Credentials: &IntegrationCredentials{APIKey: field.Some("shippo_test_...")},
}

func (*RotateAccountIntegrationCredentialsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRotateAccountIntegrationCredentialsRequest)
}

// Replaces an integration's credentials.
//
// The new credentials are checked against the provider before they're saved.
type RotateAccountIntegrationCredentialsEndpoint struct{}

func (e *RotateAccountIntegrationCredentialsEndpoint) Materialize() *apiendpoint.APIEndpoint[*RotateAccountIntegrationCredentialsRequest, *apiresource.AccountIntegration] {
	return lifecycle.Action("Rotate Integration Credentials", "/v1/settings/integrations/{id}/actions/rotate-credentials", constants.ObjectTypeAccountIntegration,
		authz.Policy{RoleType: constants.RoleTypeAdmin},
		func(svc any) apiendpoint.ServiceHandler[*RotateAccountIntegrationCredentialsRequest, *apiresource.AccountIntegration] {
			return svc.(AccountIntegrationSvc).RotateAccountIntegrationCredentials
		})
}
