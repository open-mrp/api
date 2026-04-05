package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// SwitchAccountRequest is the request to switch the user's current account.
type SwitchAccountRequest struct {
	// The ID of the account to switch to.
	AccountID string `json:"account_id" validate:"required"`
}

var sampleSwitchAccountRequest = &SwitchAccountRequest{
	AccountID: apiresource.SampleAccountID,
}

func (*SwitchAccountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSwitchAccountRequest)
}

type SwitchAccountEndpoint struct{}

func (e *SwitchAccountEndpoint) Materialize() *apiendpoint.APIEndpoint[*SwitchAccountRequest, *apiresource.Tenancy] {
	return &apiendpoint.APIEndpoint[*SwitchAccountRequest, *apiresource.Tenancy]{
		Title:             "Switch Account",
		Description:       "Switches the authenticated user's active account and returns the updated tenancy context.",
		Method:            http.MethodPut,
		Route:             "/v1/identity/me/tenancy",
		Request:           &SwitchAccountRequest{},
		Response:          &apiresource.Tenancy{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SwitchAccountRequest) (*apiresource.Tenancy, *apierror.APIError) {
			return svc.(TenancySvc).SwitchAccount
		},
	}
}
