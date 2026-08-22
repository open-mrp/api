package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to change the status of a purchase order.
type ChangePurchaseOrderStatusRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// The lifecycle transition to apply.
	//
	// - `issue`: move an `estimate` order to `issued`. Creates the order's receiving order with a line for each order line.
	// - `unissue`: move an `issued` order back to `estimate`. Deletes the receiving order.
	// - `close`: move an `issued` order to `fulfilled`. Marks the receiving order complete.
	// - `open`: move a `fulfilled` order back to `issued`. Re-opens the receiving order.
	StatusChange string `json:"status_change" validate:"required"`
	// Whether to email the purchase order to the order's contacts.
	//
	// Only applies to the `issue` action. When `true`, the purchase order submission email is sent to the order's email contacts and `acknowledgment_status` is set to `sent`. An order with no email contacts still moves to `sent` even though no email goes out.
	SendEmail bool `json:"send_email"`
}

var sampleChangePurchaseOrderStatusRequest = &ChangePurchaseOrderStatusRequest{
	StatusChange: "issue",
	SendEmail:    true,
}

func (*ChangePurchaseOrderStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleChangePurchaseOrderStatusRequest)
}

// Moves a purchase order through its lifecycle.
//
// Supported actions: `issue` (`estimate` to `issued`), `unissue` (`issued` back to `estimate`), `close` (`issued` to `fulfilled`), and `open` (`fulfilled` back to `issued`). Each action is only valid from the status noted; otherwise the request fails validation.
type ChangePurchaseOrderStatusEndpoint struct{}

func (e *ChangePurchaseOrderStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangePurchaseOrderStatusRequest, *apiresource.PurchaseOrder] {
	return (&apiendpoint.APIEndpoint[*ChangePurchaseOrderStatusRequest, *apiresource.PurchaseOrder]{
		Title:             "Change Purchase Order Status",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}/actions/change-status",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPurchaseOrders, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypePurchaseOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangePurchaseOrderStatusRequest) (*apiresource.PurchaseOrder, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).ChangePurchaseOrderStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "freight", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	})
}
