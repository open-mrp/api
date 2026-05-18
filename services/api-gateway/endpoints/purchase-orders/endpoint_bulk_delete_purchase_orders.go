package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete multiple purchase orders.
type BulkDeletePurchaseOrdersRequest struct {
	// Purchase order IDs.
	PurchaseOrderIDs []string `json:"purchase_order_ids" validate:"required"`
}

var sampleBulkDeletePurchaseOrdersRequest = &BulkDeletePurchaseOrdersRequest{
	PurchaseOrderIDs: []string{apiresource.SamplePurchaseOrderDetailID},
}

func (*BulkDeletePurchaseOrdersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkDeletePurchaseOrdersRequest)
}

// Deletes multiple purchase orders.
type BulkDeletePurchaseOrdersEndpoint struct{}

func (e *BulkDeletePurchaseOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkDeletePurchaseOrdersRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*BulkDeletePurchaseOrdersRequest, *apiresource.EmptyResource]{
		Title:             "Bulk Delete Purchase Orders",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/actions/bulk-delete",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkDeletePurchaseOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).BulkDeletePurchaseOrders
		},
	})
}
