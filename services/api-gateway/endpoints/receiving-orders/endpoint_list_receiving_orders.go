package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list receiving orders.
type ListReceivingOrdersRequest struct {
	apiresource.PaginationRequest
	// Filter by completion status.
	//
	// Completed orders are hidden when this is omitted.
	Status *constants.ReceivingOrderStatus `query:"status"`
	// Filter to orders that have at least one line for any of the given item IDs.
	ItemIDs []string `query:"item_ids"`
	// Filter to orders whose originating purchase order was placed with any of the given supplier account IDs.
	SupplierIDs []string `query:"supplier_ids"`
	// Only return orders created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"starts_at"`
	// Only return orders created on or before this date (`YYYY-MM-DD`), covering that whole day.
	EndDate *string `query:"ends_at"`
}

// Returns a paginated list of receiving orders for the current account, newest first.
//
// Only open (incomplete) orders are returned by default; pass `status` to change this.
type ListReceivingOrdersEndpoint struct{}

func (e *ListReceivingOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListReceivingOrdersRequest, *apiresource.List[apiresource.ReceivingOrder]] {
	return (&apiendpoint.APIEndpoint[*ListReceivingOrdersRequest, *apiresource.List[apiresource.ReceivingOrder]]{
		Title:             "List Receiving Orders",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainReceivingOrders, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeReceivingOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListReceivingOrdersRequest) (*apiresource.List[apiresource.ReceivingOrder], *apierror.APIError) {
			return svc.(ReceivingOrderSvc).ListReceivingOrders
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeReceivingOrder,
			Fields:     []string{"supplier", "purchase_order", "lines", "lines.order_line"},
		}),
	})
}
