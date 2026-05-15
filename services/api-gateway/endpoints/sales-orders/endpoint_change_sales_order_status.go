package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to change the status of a sales order.
type ChangeSalesOrderStatusRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Status change action (e.g., "issue", "unissue", "close", "open").
	StatusChange string `json:"status_change" validate:"required"`
	// Whether to send a notification email.
	SendEmail bool `json:"send_email"`
}

var sampleChangeSalesOrderStatusRequest = &ChangeSalesOrderStatusRequest{
	StatusChange: "issue",
	SendEmail:    true,
}

func (*ChangeSalesOrderStatusRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleChangeSalesOrderStatusRequest)
}

type ChangeSalesOrderStatusEndpoint struct{}

func (e *ChangeSalesOrderStatusEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeSalesOrderStatusRequest, *apiresource.SalesOrderDetail] {
	return &apiendpoint.APIEndpoint[*ChangeSalesOrderStatusRequest, *apiresource.SalesOrderDetail]{
		Title:             "Change Sales Order Status",
		Description:       "Changes the status of a sales order. Supported actions: issue, unissue, close, and open.",
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/actions/change-status",
		Request:           &ChangeSalesOrderStatusRequest{},
		Response:          &apiresource.SalesOrderDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeSalesOrderStatusRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).ChangeSalesOrderStatus
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "order_discount", "lines", "lines.item"},
		}),
	}
}
