package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Creates a Stripe billing portal session and returns a redirect URL for managing subscriptions.
type CreateBillingPortalSessionEndpoint struct{}

func (e *CreateBillingPortalSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.BillingPortalSessionResponse] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.BillingPortalSessionResponse]{
		Title:             "Create Billing Portal Session",
		Method:            http.MethodPost,
		Route:             "/v1/billing/portal-sessions",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeBillingPortalSessionResponse,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.BillingPortalSessionResponse, *apierror.APIError) {
			return svc.(BillingSvc).CreateBillingPortalSession
		},
	})
}
