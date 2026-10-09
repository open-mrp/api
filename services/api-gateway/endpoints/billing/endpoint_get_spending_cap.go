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

// Returns the monthly cap on agent spending for the account.
//
// The cap limits estimated agent LLM spend within a billing month; Get Account Usage reports how much of it has been spent so far.
//
// Requires `billing:read`. Customer and supplier portal users are refused with `403`.
type GetSpendingCapEndpoint struct{}

func (e *GetSpendingCapEndpoint) Materialize() *apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.SpendingCapResponse] {
	return (&apiendpoint.APIEndpoint[*apiresource.EmptyResource, *apiresource.SpendingCapResponse]{
		Title:               "Get Spending Cap",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/billing/spending-cap",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBilling, Action: types.ActionRead}},
		ObjectType:          constants.ObjectTypeSpendingCapResponse,
		Extras:              apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.SpendingCapResponse, *apierror.APIError) {
			return svc.(BillingSvc).GetSpendingCap
		},
	})
}
