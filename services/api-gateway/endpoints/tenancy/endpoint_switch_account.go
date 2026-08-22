package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to switch the authenticated user's active account.
type SwitchAccountRequest struct {
	// ID of the account to switch to.
	AccountID string `json:"account_id" validate:"required"`
}

var sampleSwitchAccountRequest = &SwitchAccountRequest{
	AccountID: apiresource.SampleAccountID,
}

func (*SwitchAccountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSwitchAccountRequest)
}

// Switches the authenticated user's active account and returns the updated tenancy context.
//
// The user must have access to the requested account; switching to an account where the user is disabled or removed, or to a suspended or deactivated account, is rejected. Switching into a sandbox is also rejected when the user is disabled or removed on the production account that owns it.
//
// A successful switch marks the account as the user's most recently used one, which Get Tenancy takes into account when it picks a current account for a request that does not target one.
type SwitchAccountEndpoint struct{}

func (e *SwitchAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*SwitchAccountRequest, *apiresource.Tenancy] {
	return (&apiendpoint.APIEndpoint[*SwitchAccountRequest, *apiresource.Tenancy]{
		Title:             "Switch Account",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/identity/me/tenancy",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeTenancy,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SwitchAccountRequest) (*apiresource.Tenancy, *apierror.APIError) {
			return svc.(TenancySvc).SwitchAccount
		},
		Extras: apiendpoint.APIEndpointExtras{
			HideFromRequestLog: true,
		},
	})
}
