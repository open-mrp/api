package tenancyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to switch the authenticated user's active account.
type SwitchAccountRequest struct {
	// Account ID.
	AccountID string `json:"account_id" validate:"required"`
}

var sampleSwitchAccountRequest = &SwitchAccountRequest{
	AccountID: apiresource.SampleAccountID,
}

func (*SwitchAccountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSwitchAccountRequest)
}

// Switches the authenticated user's active account and returns the updated tenancy context.
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
			SkipRequestLogging: true,
		},
	})
}
