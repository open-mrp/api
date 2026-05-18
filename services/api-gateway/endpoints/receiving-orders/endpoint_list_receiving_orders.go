package receivingorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list receiving orders.
type ListReceivingOrdersRequest struct {
	apiresource.PaginationRequest
	// Filter by status.
	Status *string `query:"status"`
	// Filter by item IDs present in receiving order lines.
	ItemIDs []string `query:"item_ids"`
	// Filter by supplier account IDs.
	SupplierIDs []string `query:"supplier_ids"`
	// Filter by start date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter by end date (inclusive).
	EndDate *string `query:"end_date"`
}

// TODO: stop returning ReceivingOrderSummary; return the full ReceivingOrder apiresource and use proper includes values to control expansion.

// Returns a paginated list of receiving orders for the current account.
type ListReceivingOrdersEndpoint struct{}

func (e *ListReceivingOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListReceivingOrdersRequest, *apiresource.List[apiresource.ReceivingOrderSummary]] {
	return (&apiendpoint.APIEndpoint[*ListReceivingOrdersRequest, *apiresource.List[apiresource.ReceivingOrderSummary]]{
		Title:             "List Receiving Orders",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/receiving-orders",
		Request:           &ListReceivingOrdersRequest{},
		Response:          &apiresource.List[apiresource.ReceivingOrderSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListReceivingOrdersRequest) (*apiresource.List[apiresource.ReceivingOrderSummary], *apierror.APIError) {
			return svc.(ReceivingOrderSvc).ListReceivingOrders
		},
	}).WithDocSource(e)
}
