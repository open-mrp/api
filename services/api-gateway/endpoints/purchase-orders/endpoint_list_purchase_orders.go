package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list purchase orders.
type ListPurchaseOrdersRequest struct {
	apiresource.PaginationRequest
	// Filter by status codes.
	StatusCodes []string `query:"status_codes"`
	// Filter by item IDs.
	ItemIDs []string `query:"item_ids"`
	// Filter by supplier IDs.
	SupplierIDs []string `query:"supplier_ids"`
	// Filter by start date (inclusive).
	StartDate *string `query:"start_date"`
	// Filter by end date (inclusive).
	EndDate *string `query:"end_date"`
}

// TODO: stop returning PurchaseOrderSummary; return the full PurchaseOrder apiresource and use proper includes values to control expansion.
type ListPurchaseOrdersEndpoint struct{}

func (e *ListPurchaseOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPurchaseOrdersRequest, *apiresource.List[apiresource.PurchaseOrderSummary]] {
	return &apiendpoint.APIEndpoint[*ListPurchaseOrdersRequest, *apiresource.List[apiresource.PurchaseOrderSummary]]{
		Title:             "List Purchase Orders",
		Description:       "Returns a paginated list of purchase orders for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders",
		Request:           &ListPurchaseOrdersRequest{},
		Response:          &apiresource.List[apiresource.PurchaseOrderSummary]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPurchaseOrdersRequest) (*apiresource.List[apiresource.PurchaseOrderSummary], *apierror.APIError) {
			return svc.(PurchaseOrderSvc).ListPurchaseOrders
		},
	}
}
