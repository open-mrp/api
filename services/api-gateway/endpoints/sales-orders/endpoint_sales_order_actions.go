package salesorderep

import (
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	"github.com/open-mrp/api/services/api-gateway/pkg/lifecycle"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

var salesOrderUpdatePolicy = authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate}}}

// Request to issue a sales order.
type IssueSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Whether to email the order acknowledgment, with a PDF of the order, to the order's acknowledgment recipients.
	//
	// `acknowledgment_status` becomes `sent` once the email is queued, shortly after this call returns. An order with no acknowledgment recipients sends nothing.
	NotifyCustomer bool `json:"notify_customer"`
}

var sampleIssueSalesOrderRequest = &IssueSalesOrderRequest{NotifyCustomer: true}

func (*IssueSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleIssueSalesOrderRequest)
}

// Request to unissue a sales order.
type UnissueSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// Request to cancel a sales order.
type CancelSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// Issues the order: creates its pick and reserves inventory.
//
// Moves a `draft` order to `issued` and stamps its ship-by date. Any other status is a 409 `invalid_state_transition`.
type IssueSalesOrderEndpoint struct{}

func (*IssueSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*IssueSalesOrderRequest, *apiresource.SalesOrder] {
	e := lifecycle.Action("Issue Sales Order", "/v1/sales/sales-orders/{id}/actions/issue", constants.ObjectTypeSalesOrder, salesOrderUpdatePolicy,
		func(svc any) apiendpoint.ServiceHandler[*IssueSalesOrderRequest, *apiresource.SalesOrder] {
			return svc.(SalesOrderSvc).IssueSalesOrder
		})
	e.IncludeConfig = salesOrderIncludeConfig()
	return e
}

// Returns it to `draft`: deletes the pick and releases inventory.
//
// Only an `issued` order can be unissued; any other status is a 409 `invalid_state_transition`. Its ship-by date is cleared.
type UnissueSalesOrderEndpoint struct{}

func (*UnissueSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnissueSalesOrderRequest, *apiresource.SalesOrder] {
	e := lifecycle.Action("Unissue Sales Order", "/v1/sales/sales-orders/{id}/actions/unissue", constants.ObjectTypeSalesOrder, salesOrderUpdatePolicy,
		func(svc any) apiendpoint.ServiceHandler[*UnissueSalesOrderRequest, *apiresource.SalesOrder] {
			return svc.(SalesOrderSvc).UnissueSalesOrder
		})
	e.IncludeConfig = salesOrderIncludeConfig()
	return e
}

// Stops the order: closes whatever hasn't shipped and releases its inventory.
//
// Becomes `canceled` if nothing shipped, otherwise `fulfilled`. Only an `issued` order can be canceled (409 `invalid_state_transition` otherwise); delete a `draft` instead.
type CancelSalesOrderEndpoint struct{}

func (*CancelSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CancelSalesOrderRequest, *apiresource.SalesOrder] {
	e := lifecycle.Action("Cancel Sales Order", "/v1/sales/sales-orders/{id}/actions/cancel", constants.ObjectTypeSalesOrder, salesOrderUpdatePolicy,
		func(svc any) apiendpoint.ServiceHandler[*CancelSalesOrderRequest, *apiresource.SalesOrder] {
			return svc.(SalesOrderSvc).CancelSalesOrder
		})
	e.IncludeConfig = salesOrderIncludeConfig()
	return e
}
