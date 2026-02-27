package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const ensureBillingCustomerDescription string = `Links or fetches a Stripe customer for the requesting account.
If a Stripe customer already exists it is returned; otherwise a new one is created.`

type EnsureBillingCustomerEndpoint struct{}

func (e *EnsureBillingCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnsureBillingCustomerResponse] {
	return &apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnsureBillingCustomerResponse]{
		Title:             "Ensure Billing Customer",
		Description:       ensureBillingCustomerDescription,
		Method:            http.MethodPut,
		Route:             "/v1/billing/accounts",
		ContentType:       "application/json",
		Request:           &apiresource.EmptyResource{},
		Response:          apiresource.SampleEnsureBillingCustomerResponse,
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.EnsureBillingCustomerResponse, *apierror.APIError) {
			return svc.(BillingSvc).EnsureBillingCustomer
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
