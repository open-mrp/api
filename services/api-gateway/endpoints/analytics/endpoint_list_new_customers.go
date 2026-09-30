package analyticsep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// ListNewCustomersRequest is the request to list the customers added in a period that have ordered.
type ListNewCustomersRequest struct {
	// Start of the period, by when the customer was added, inclusive.
	StartDate time.Time `json:"starts_at" validate:"required"`
	// End of the period, by when the customer was added, inclusive.
	EndDate time.Time `json:"ends_at" validate:"required"`
	// Only customers in any of these customer groups, as their group or one of their price groups.
	CustomerGroupIDs []string `json:"customer_group_ids,omitzero"`
	// Only customers whose default sales rep is one of these account users.
	SalesRepIDs []string `json:"sales_rep_ids,omitzero"`
	// Opaque cursor from a previous page's `next_page_url` or `previous_page_url`. Omit for the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of customers to return.
	Limit int32 `query:"limit" default:"100" validate:"min=1,max=1000"`
}

// Returns the customers added in a period that have placed an order, newest first order first, each with the date of its first order and its lifetime sales. Sales count sales orders only: lines priced above zero, outside the shipping and misc product lines, over the customer's whole history rather than the period. Customers that have never ordered are left out.
type ListNewCustomersEndpoint struct{}

func (e *ListNewCustomersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListNewCustomersRequest, *apiresource.List[apiresource.NewCustomer]] {
	return (&apiendpoint.APIEndpoint[*ListNewCustomersRequest, *apiresource.List[apiresource.NewCustomer]]{
		Title:               "List New Customers",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/new-customers-table",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ReadOnly:            true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainCustomers, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListNewCustomersRequest) (*apiresource.List[apiresource.NewCustomer], *apierror.APIError) {
			return svc.(AnalyticsSvc).ListNewCustomers
		},
	})
}
