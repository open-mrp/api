package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const getAccountUsageDescription string = `Returns current resource usage (seats, invoices, batches, sandboxes) with plan limits and subscription info for the authenticated account.`

type GetAccountUsageEndpoint struct{}

func (e *GetAccountUsageEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.AccountUsageResponse] {
	return &apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.AccountUsageResponse]{
		Title:             "Get Account Usage",
		Description:       getAccountUsageDescription,
		Method:            http.MethodGet,
		Route:             "/v1/billing/accounts/usage",
		Request:           &apiresource.EmptyResource{},
		Response:          apiresource.SampleAccountUsageResponse,
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.AccountUsageResponse, *apierror.APIError) {
			return svc.(BillingSvc).GetAccountUsage
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
