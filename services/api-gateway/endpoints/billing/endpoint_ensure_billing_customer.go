package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Ensures a Stripe billing customer exists for the account.
type EnsureBillingCustomerEndpoint struct{}

func (e *EnsureBillingCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnsureBillingCustomerResponse] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnsureBillingCustomerResponse]{
		Title:             "Ensure Billing Customer",
		Method:            http.MethodPut,
		Route:             "/v1/billing/accounts",
		ContentType:       "application/json",
		Request:           &apiresource.EmptyResource{},
		Response:          &apiresource.EnsureBillingCustomerResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.EnsureBillingCustomerResponse, *apierror.APIError) {
			return svc.(BillingSvc).EnsureBillingCustomer
		},
	}).WithDocSource(e)
}
