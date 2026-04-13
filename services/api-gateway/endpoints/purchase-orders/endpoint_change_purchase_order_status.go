package purchaseorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to change the status of a purchase order.
type ChangePurchaseOrderStatusRequest struct {
	// Purchase order ID.
	PurchaseOrderID string `path:"id" validate:"required"`
	// Status change action (e.g., "issue", "unissue", "close", "open").
	StatusChange string `json:"status_change" validate:"required"`
	// Whether to send a notification email.
	SendEmail bool `json:"send_email"`
}

var sampleChangePurchaseOrderStatusRequest = &ChangePurchaseOrderStatusRequest{
	StatusChange: "issue",
	SendEmail:    true,
}

func (*ChangePurchaseOrderStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleChangePurchaseOrderStatusRequest)
}

type ChangePurchaseOrderStatusEndpoint struct{}

func (e *ChangePurchaseOrderStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangePurchaseOrderStatusRequest, *apiresource.PurchaseOrderDetail] {
	return &apiendpoint.APIEndpoint[*ChangePurchaseOrderStatusRequest, *apiresource.PurchaseOrderDetail]{
		Title:             "Change Purchase Order Status",
		Description:       "Changes the status of a purchase order. Supported actions: issue (draft to issued), unissue (issued to draft), close (issued to closed), open (closed to issued).",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/operations/purchase-orders/{id}/actions/change-status",
		Request:           &ChangePurchaseOrderStatusRequest{},
		Response:          &apiresource.PurchaseOrderDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangePurchaseOrderStatusRequest) (*apiresource.PurchaseOrderDetail, *apierror.APIError) {
			return svc.(PurchaseOrderSvc).ChangePurchaseOrderStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePurchaseOrder,
			Fields:     []string{"supplier", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "receiving_order", "lines", "contacts"},
		}),
	}
}
