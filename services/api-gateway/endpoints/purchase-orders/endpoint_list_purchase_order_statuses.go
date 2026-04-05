package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListPurchaseOrderStatusesRequest is the request to list purchase order statuses.
type ListPurchaseOrderStatusesRequest struct {
	apiresource.PaginationRequest
}

type ListPurchaseOrderStatusesEndpoint struct{}

func (e *ListPurchaseOrderStatusesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPurchaseOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]] {
	return &apiendpoint.APIEndpoint[*ListPurchaseOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]]{
		Title:             "List Purchase Order Statuses",
		Description:       "Returns a paginated list of available purchase order status values.",
		Method:            http.MethodGet,
		Route:             "/v1/operations/purchase-orders/statuses",
		Request:           &ListPurchaseOrderStatusesRequest{},
		Response:          &apiresource.List[apiresource.SalesOrderStatus]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPurchaseOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError) {
			return svc.(PurchaseOrderSvc).ListPurchaseOrderStatuses
		},
	}
}
