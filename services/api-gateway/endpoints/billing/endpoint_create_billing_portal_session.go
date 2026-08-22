package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Creates a Stripe billing portal session for the account and returns the URL to send an admin to.
//
// The portal is where the account manages payment methods, invoices, and its subscription directly in Stripe. The account must already have a Stripe customer; create one with Ensure Billing Customer first.
type CreateBillingPortalSessionEndpoint struct{}

func (e *CreateBillingPortalSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.BillingPortalSessionResponse] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.BillingPortalSessionResponse]{
		Title:             "Create Billing Portal Session",
		Method:            http.MethodPost,
		Route:             "/v1/billing/portal-sessions",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeBillingPortalSessionResponse,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.BillingPortalSessionResponse, *apierror.APIError) {
			return svc.(BillingSvc).CreateBillingPortalSession
		},
	})
}
