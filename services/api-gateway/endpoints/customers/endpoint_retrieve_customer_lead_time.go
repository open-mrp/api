package customerep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to resolve a customer's ship-by lead time.
type RetrieveCustomerLeadTimeRequest struct {
	// Customer ID.
	CustomerID string `path:"id" validate:"required"`
}

// Retrieves a customer's lead time.
//
// The lead time a new order for the customer would be committed to, and where it came from: the customer's own `defaults.lead_time`, then its parent customer's, then its group's, then your account default. Changing a lead time moves what future orders promise; orders already issued keep their ship-by dates.
type RetrieveCustomerLeadTimeEndpoint struct{}

func (e *RetrieveCustomerLeadTimeEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveCustomerLeadTimeRequest, *apiresource.CustomerLeadTime] {
	return (&apiendpoint.APIEndpoint[*RetrieveCustomerLeadTimeRequest, *apiresource.CustomerLeadTime]{
		Title:             "Retrieve Customer Lead Time",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/customers/{id}/lead-time",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeCustomerLeadTime,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveCustomerLeadTimeRequest) (*apiresource.CustomerLeadTime, *apierror.APIError) {
			return svc.(CustomerSvc).GetCustomerLeadTime
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeCustomerLeadTime,
			Fields:     []string{"customer", "account_group", "parent_customer"},
		}),
	})
}
