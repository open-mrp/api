package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Returns resource usage for the account, including seats, invoices, batches, sandboxes, and subscription details.
type GetAccountUsageEndpoint struct{}

func (e *GetAccountUsageEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.AccountUsageResponse] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.AccountUsageResponse]{
		Title:             "Get Account Usage",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/billing/accounts/usage",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeAccountUsageResponse,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.AccountUsageResponse, *apierror.APIError) {
			return svc.(BillingSvc).GetAccountUsage
		},
	})
}
