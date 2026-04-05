package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// BulkDeletePurchaseOrdersRequest is the request to delete multiple purchase orders.
type BulkDeletePurchaseOrdersRequest struct {
	// The IDs of the purchase orders to delete.
	PurchaseOrderIDs []string `json:"purchase_order_ids" validate:"required"`
}

var sampleBulkDeletePurchaseOrdersRequest = &BulkDeletePurchaseOrdersRequest{
	PurchaseOrderIDs: []string{apiresource.SamplePurchaseOrderDetailID},
}

func (*BulkDeletePurchaseOrdersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkDeletePurchaseOrdersRequest)
}

type BulkDeletePurchaseOrdersEndpoint struct{}

func (e *BulkDeletePurchaseOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkDeletePurchaseOrdersRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*BulkDeletePurchaseOrdersRequest, *apiresource.EmptyResource]{
		Title:             "Bulk Delete Purchase Orders",
		Description:       "Deletes multiple purchase orders in a single operation.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/purchase-orders/actions/bulk-delete",
		Request:           &BulkDeletePurchaseOrdersRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkDeletePurchaseOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).BulkDeletePurchaseOrders
		},
	}
}
