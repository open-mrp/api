package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Ensures a Stripe billing customer exists for the account, creating one if necessary.
type EnsureBillingCustomerEndpoint struct{}

func (e *EnsureBillingCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnsureBillingCustomerResponse] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.EnsureBillingCustomerResponse]{
		Title:             "Ensure Billing Customer",
		Method:            http.MethodPut,
		Route:             "/v1/billing/accounts",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		RequiredRoleType:  constants.RoleTypeAdmin,
		Preview:           true,
		ObjectType:        constants.ObjectTypeEnsureBillingCustomerResponse,
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.EnsureBillingCustomerResponse, *apierror.APIError) {
			return svc.(BillingSvc).EnsureBillingCustomer
		},
	})
}
