package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list sales orders.
type ListSalesOrdersRequest struct {
	apiresource.PaginationRequest
	// Filter by status codes.
	StatusCodes []string `query:"status_codes"`
	// Filter by item IDs.
	ItemIDs []string `query:"item_ids"`
	// Filter by product line IDs.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by customer IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by customer group IDs.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Filter by sales rep IDs.
	SalesRepIDs []string `query:"sales_rep_ids"`
	// Filter by start date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter by end date (inclusive).
	EndDate *string `query:"end_date"`
	// Whether to exclude internal orders.
	ExcludeInternalOrders bool `query:"exclude_internal_orders"`
}

// TODO: stop returning SalesOrderSummary; return the full SalesOrder apiresource and use proper includes values to control expansion.
type ListSalesOrdersEndpoint struct{}

func (e *ListSalesOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesOrdersRequest, *apiresource.List[apiresource.SalesOrderSummary]] {
	return &apiendpoint.APIEndpoint[*ListSalesOrdersRequest, *apiresource.List[apiresource.SalesOrderSummary]]{
		Title:             "List Sales Orders",
		Description:       "Returns a paginated list of sales orders for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders",
		Request:           &ListSalesOrdersRequest{},
		Response:          &apiresource.List[apiresource.SalesOrderSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesOrdersRequest) (*apiresource.List[apiresource.SalesOrderSummary], *apierror.APIError) {
			return svc.(SalesOrderSvc).ListSalesOrders
		},
	}
}
