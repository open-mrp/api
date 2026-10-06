package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list sales orders.
type ListSalesOrdersRequest struct {
	apiresource.PaginationRequest
	// Restricts results to orders in any of these lifecycle statuses.
	StatusCodes []constants.SalesOrderStatusCode `query:"status_codes"`
	// Restricts results to orders that have at least one line for any of these inventory items.
	ItemIDs []string `query:"item_ids"`
	// Restricts results to orders that have at least one line whose product belongs to any of these product lines.
	ProductLineIDs []string `query:"product_line_ids"`
	// Restricts results to orders placed by any of these customers.
	CustomerIDs []string `query:"customer_ids"`
	// Restricts results to orders placed by customers belonging to any of these account groups.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Restricts results to orders credited to any of these sales reps.
	//
	// These are account user IDs, matching the `sales_rep` on the order.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Only include orders created on or after this date (`YYYY-MM-DD`, UTC). A full timestamp (RFC 3339) is also accepted, to bound the range at a local midnight.
	StartDate *string `query:"starts_at" validate:"omitempty,date_filter"`
	// Only include orders created at or before the start of this date (`YYYY-MM-DD`, UTC), so orders created later on the end date itself are excluded; pass the following day to include them. A full timestamp (RFC 3339) is also accepted and used as given.
	EndDate *string `query:"ends_at" validate:"omitempty,date_filter"`
	// Earliest ship-by date to include, in `YYYY-MM-DD` format. Inclusive of the date itself.
	ShipByAfter *string `query:"ship_by_after" validate:"omitempty,date_filter"`
	// Latest ship-by date to include, in `YYYY-MM-DD` format. Inclusive of the date itself.
	ShipByBefore *string `query:"ship_by_before" validate:"omitempty,date_filter"`
	// Restricts results to orders that are, or are not, past their ship-by date.
	//
	// An order is past due when it is still `issued` and its ship-by date has passed. A fulfilled order that shipped late is not past due — it is delivered, and how late it was is a delivery-performance question rather than a backlog one.
	PastDue *bool `query:"past_due"`
}

// Returns a paginated list of sales orders for the current account, newest first.
//
// A free-text search term (`q`) is matched as an exact value against the order number and the customer purchase order number, and still respects the other filters. Customer accounts calling this endpoint only ever see their own orders.
type ListSalesOrdersEndpoint struct{}

func (e *ListSalesOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesOrdersRequest, *apiresource.List[apiresource.SalesOrder]] {
	return (&apiendpoint.APIEndpoint[*ListSalesOrdersRequest, *apiresource.List[apiresource.SalesOrder]]{
		Title:                   "List Sales Orders",
		Method:                  http.MethodGet,
		ContentType:             "application/json",
		Route:                   "/v1/sales/sales-orders",
		SuccessStatusCode:       http.StatusOK,
		Public:                  true,
		Preview:                 true,
		AgentTool:               true,
		RequiredPermissions:     []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}},
		CounterpartyPermissions: apiendpoint.Counterparties(types.ActionRead),
		ObjectType:              constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrder], *apierror.APIError) {
			return svc.(SalesOrderSvc).ListSalesOrders
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "sales_rep", "created_by", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "order_discount", "totals", "contacts", "related.pick", "related.production_run", "related.shipments", "related.invoices", "lines", "lines.product", "lines.product.item", "lines.product.item.category", "lines.product.item.category.properties", "lines.product.item.category.unit_group", "lines.product.item.category.unit_group.base_unit", "lines.product.item.category.unit_group.associated_units", "lines.product.item.category.unit_group.associated_units.unit", "lines.product.product_line", "lines.quantity_ordered", "lines.quantity_ordered.unit", "lines.unit_price", "lines.unit_price.numerator_unit", "lines.unit_price.denominator_unit", "lines.unit_cost", "lines.unit_cost.numerator_unit", "lines.unit_cost.denominator_unit", "lines.totals"},
		}),
	})
}
