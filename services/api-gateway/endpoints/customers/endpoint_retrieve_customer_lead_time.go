package customerep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to resolve a customer's ship-by lead time.
type RetrieveCustomerLeadTimeRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

// Returns the ship-by lead time a new order for this customer would be committed to.
//
// Resolved through the same chain the issue path stamps onto an order, most specific first: a lead time set on the customer, then on its parent account, then on the customer's account group, then the account-wide default. `source` names which rule applied, so a form can show where the number came from rather than leaving a rep to guess.
//
// A lead time set on a parent account therefore governs every child account under it that has not set its own, which is how a head office's terms are given to its locations without repeating them on each one.
//
// This is a preview of a commitment, not the commitment itself. An order takes its own `ship_by_date` when it is issued and keeps it afterwards, so changing a lead time here moves what future orders will promise and leaves promises already made alone.
type RetrieveCustomerLeadTimeEndpoint struct{}

func (e *RetrieveCustomerLeadTimeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveCustomerLeadTimeRequest, *apiresource.CustomerLeadTime] {
	return (&apiendpoint.APIEndpoint[*RetrieveCustomerLeadTimeRequest, *apiresource.CustomerLeadTime]{
		Title:               "Retrieve Customer Lead Time",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/sales/customers/{id}/lead-time",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeCustomerLeadTime,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCustomerLeadTimeRequest) (*apiresource.CustomerLeadTime, *apierror.APIError) {
			return svc.(CustomerSvc).GetCustomerLeadTime
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomerLeadTime,
			Fields:     []string{"account_group", "parent_customer"},
		}),
	})
}
