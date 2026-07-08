package billingep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Returns the monthly agent spending cap for the account.
//
// A null `cap_cents` means no cap is set.
type GetSpendingCapEndpoint struct{}

func (e *GetSpendingCapEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.SpendingCapResponse] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.SpendingCapResponse]{
		Title:             "Get Spending Cap",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/billing/spending-cap",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		// Account-scoped read; mirrors retrieve_account's account:read (self:read) gate.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainAccount, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeSpendingCapResponse,
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.SpendingCapResponse, *apierror.APIError) {
			return svc.(BillingSvc).GetSpendingCap
		},
	})
}
