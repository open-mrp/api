package salesorderep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list sales orders.
type ListSalesOrdersRequest struct {
	object.ListRequest
	apirequest.CreatedRange
	// Exact match on `number` or `customer_purchase_order_number`.
	Query *string `query:"q" validate:"omitempty,max=500"`
	// Only return orders with any of these statuses: `draft`, `issued`, `fulfilled` or `canceled`.
	Statuses []constants.SalesOrderStatus `query:"statuses"`
	// Only return orders with any of these payment statuses: `unpaid`, `partially_paid` or `paid`.
	PaymentStatuses []constants.SalesOrderPaymentStatus `query:"payment_statuses"`
	// Only return orders with a line for any of these items.
	ItemIDs []string `query:"item_ids"`
	// Only return orders with a line for an item in any of these product lines.
	ProductLineIDs []string `query:"product_line_ids"`
	// Only return orders for any of these customers.
	CustomerIDs []string `query:"customer_ids"`
	// Only return orders for customers in any of these account groups.
	GroupIDs []string `query:"group_ids"`
	// Only return orders credited to any of these sales reps (account user IDs).
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Earliest ship-by date to include, as `YYYY-MM-DD`. Inclusive.
	ShipByAfter *string `query:"ship_by_after" validate:"omitempty,date_filter"`
	// Latest ship-by date to include, as `YYYY-MM-DD`. Inclusive.
	ShipByBefore *string `query:"ship_by_before" validate:"omitempty,date_filter"`
	// Only return orders that are, or are not, past their ship-by date.
	//
	// An order is past due while it is `issued` and its ship-by date has passed. A fulfilled order that shipped late is not past due.
	PastDue *bool `query:"past_due"`
}

// Returns a list of sales orders.
//
// Sorted by creation date, newest first. Customer and supplier portal users only see the orders their own account placed.
type ListSalesOrdersEndpoint struct{}

func (e *ListSalesOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesOrdersRequest, *apiresource.List[apiresource.SalesOrder]] {
	return (&apiendpoint.APIEndpoint[*ListSalesOrdersRequest, *apiresource.List[apiresource.SalesOrder]]{
		Title:             "List Sales Orders",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrder], *apierror.APIError) {
			return svc.(SalesOrderSvc).ListSalesOrders
		},
		IncludeConfig: salesOrderIncludeConfig(),
	})
}
