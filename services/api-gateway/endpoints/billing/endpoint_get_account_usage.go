package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Returns the account's resource usage against its plan limits, along with subscription status, plan pricing, and estimated agent spending.
//
// Seats and sandboxes are current totals, while invoices and batches are counted from the start of the current billing period. The plan name and base fee come from the pricing plan configured in Stripe, so they can differ from the name and price the same plan advertises on the pricing page.
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
		// Account-scoped read; mirrors retrieve_account's account:read (self:read) gate.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeAccountUsageResponse,
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.AccountUsageResponse, *apierror.APIError) {
			return svc.(BillingSvc).GetAccountUsage
		},
	})
}
