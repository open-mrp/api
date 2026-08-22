package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
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

// Deletes multiple purchase orders, each along with its lines, email contacts, and receiving order.
//
// The whole request is all-or-nothing: if any ID cannot be found in your account or refers to an order in `fulfilled` status, nothing is deleted.
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
