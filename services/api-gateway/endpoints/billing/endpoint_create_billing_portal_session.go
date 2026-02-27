package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const createBillingPortalSessionDescription string = `Creates a Stripe billing portal session for managing subscriptions.
Returns a URL that can be used to redirect the user to the Stripe billing portal.`

type CreateBillingPortalSessionEndpoint struct{}

func (e *CreateBillingPortalSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.BillingPortalSessionResponse] {
	return &apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.BillingPortalSessionResponse]{
		Title:             "Create Billing Portal Session",
		Description:       createBillingPortalSessionDescription,
		Method:            http.MethodPost,
		Route:             "/v1/billing/portal-sessions",
		ContentType:       "application/json",
		Request:           &apiresource.EmptyResource{},
		Response:          apiresource.SampleBillingPortalSessionResponse,
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.BillingPortalSessionResponse, *apierror.APIError) {
			return svc.(BillingSvc).CreateBillingPortalSession
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
