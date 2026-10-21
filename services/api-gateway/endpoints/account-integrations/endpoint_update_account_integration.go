package accountintegrationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
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
	// Most the provider may spend for the account in a UTC calendar month, or `null` to remove the limit. Only `openrouter` takes a limit, in `usd`.
	MonthlySpendLimit field.Clearable[apirequest.MoneyInput] `json:"monthly_spend_limit,omitzero"`
}

var sampleUpdateAccountIntegrationRequest = &UpdateAccountIntegrationRequest{
	Name:   field.Some("Updated Stripe Integration"),
	Status: field.Some(constants.AccountIntegrationStatusActive),
}

func (*UpdateAccountIntegrationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateAccountIntegrationRequest)
}

// Renames an account integration, activates or deactivates it, or changes its monthly spend limit.
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
