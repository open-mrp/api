package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete multiple purchase orders.
type BulkDeletePurchaseOrdersRequest struct {
	// IDs of the purchase orders to delete.
	PurchaseOrderIDs []string `json:"purchase_order_ids" validate:"required"`
}

var sampleBulkDeletePurchaseOrdersRequest = &BulkDeletePurchaseOrdersRequest{
	PurchaseOrderIDs: []string{apiresource.SamplePurchaseOrderID},
}

func (*BulkDeletePurchaseOrdersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkDeletePurchaseOrdersRequest)
}

// Deletes multiple purchase orders in a single request.
//
// If any of the orders is in `fulfilled` status the request fails and no orders are deleted.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkDeletePurchaseOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).BulkDeletePurchaseOrders
		},
	})
}
