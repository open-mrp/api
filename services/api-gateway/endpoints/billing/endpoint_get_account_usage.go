package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type GetAccountUsageEndpoint struct{}

func (e *GetAccountUsageEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.AccountUsageResponse] {
	return &apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.AccountUsageResponse]{
		Title:             "Get Account Usage",
		Description:       "Returns current resource usage for the account, including seats, invoices, batches, and sandboxes, along with plan limits and subscription details.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/billing/accounts/usage",
		Request:           &apiresource.EmptyResource{},
		Response:          &apiresource.AccountUsageResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.AccountUsageResponse, *apierror.APIError) {
			return svc.(BillingSvc).GetAccountUsage
		},
	}
}
