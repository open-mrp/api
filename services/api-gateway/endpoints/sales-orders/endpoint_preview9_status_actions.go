package salesorderep

import (
	"context"
	"net/http"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// salesOrderStatusActionsRemoved points a forge caller of the preview.9 status routes at their replacements.
const salesOrderStatusActionsRemoved = "Use POST /v1/sales/sales-orders/{id}/actions/issue, /unissue or /cancel."

// preview9StatusAction is a preview.9 sales order status route: a PUT to the action's path, answering 1.0.forge-preview.9
// callers only.
func preview9StatusAction[TReq any](title, verb string, handler func(svc any) func(context.Context, TReq) (*apiresource.SalesOrder, *apierror.APIError)) *apiendpoint.APIEndpoint[TReq, *apiresource.SalesOrder] {
	return &apiendpoint.APIEndpoint[TReq, *apiresource.SalesOrder]{
		Title:             title,
		Method:            http.MethodPut,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/actions/" + verb,
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Auth:              salesOrderUpdatePolicy,
		ObjectType:        constants.ObjectTypeSalesOrder,
		ServiceHandler:    handler,
	}
}

// Issues a sales order, moving it from `estimate` to `issued`.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge issues with a POST.
type Preview9IssueSalesOrderEndpoint struct{}

func (*Preview9IssueSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*IssueSalesOrderRequest, *apiresource.SalesOrder] {
	return preview9StatusAction("Issue Sales Order (preview.9)", "issue", func(svc any) func(context.Context, *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
		return svc.(SalesOrderSvc).Preview9IssueSalesOrder
	})
}

// Unissues a sales order, moving it from `issued` back to `estimate`.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge unissues with a POST.
type Preview9UnissueSalesOrderEndpoint struct{}

func (*Preview9UnissueSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnissueSalesOrderRequest, *apiresource.SalesOrder] {
	return preview9StatusAction("Unissue Sales Order (preview.9)", "unissue", func(svc any) func(context.Context, *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
		return svc.(SalesOrderSvc).Preview9UnissueSalesOrder
	})
}

// Closes a sales order, moving it from `issued` to `fulfilled` and closing whatever hasn't shipped.
//
// Kept for 1.0.forge-preview.9 callers only; 1.0.forge cancels instead.
type Preview9CloseSalesOrderEndpoint struct{}

func (*Preview9CloseSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelSalesOrderRequest, *apiresource.SalesOrder] {
	return preview9StatusAction("Close Sales Order (preview.9)", "close", func(svc any) func(context.Context, *CancelSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
		return svc.(SalesOrderSvc).Preview9CloseSalesOrder
	})
}

// Reopens a fulfilled sales order, moving it back to `issued`.
//
// Kept for 1.0.forge-preview.9 callers only; on 1.0.forge an order returns to `issued` when a shipment is unshipped.
type Preview9OpenSalesOrderEndpoint struct{}

func (*Preview9OpenSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelSalesOrderRequest, *apiresource.SalesOrder] {
	return preview9StatusAction("Open Sales Order (preview.9)", "open", func(svc any) func(context.Context, *CancelSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
		return svc.(SalesOrderSvc).Preview9OpenSalesOrder
	})
}
