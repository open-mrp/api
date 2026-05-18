package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list purchase order statuses.
type ListPurchaseOrderStatusesRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of available purchase order status values.
type ListPurchaseOrderStatusesEndpoint struct{}

func (e *ListPurchaseOrderStatusesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPurchaseOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]] {
	return (&apiendpoint.APIEndpoint[*ListPurchaseOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]]{
		Title:             "List Purchase Order Statuses",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/statuses",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPurchaseOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError) {
			return svc.(PurchaseOrderSvc).ListPurchaseOrderStatuses
		},
	})
}
