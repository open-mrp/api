package salesorderep

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

// Request to issue a sales order.
type IssueSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Whether to notify the customer.
	//
	// When `true`, an order acknowledgement email with a PDF of the order is sent to the acknowledgement contacts on the order and the order's `acknowledgment_status` becomes `sent`. An order with no acknowledgement contacts sends nothing and leaves its `acknowledgment_status` unchanged.
	NotifyCustomer bool `json:"notify_customer"`
}

var sampleIssueSalesOrderRequest = &IssueSalesOrderRequest{NotifyCustomer: true}

func (*IssueSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleIssueSalesOrderRequest)
}

// Issues a sales order, transitioning it from `estimate` to `issued`.
//
// Issuing commits the order for fulfillment: a pick is created for the order's sale lines and inventory is reserved for each line tied to an inventory item. Only an order still in `estimate` can be issued.
type IssueSalesOrderEndpoint struct{}

func (e *IssueSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*IssueSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*IssueSalesOrderRequest, *apiresource.SalesOrder]{
		Title:               "Issue Sales Order",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/{id}/actions/issue",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).IssueSalesOrder
		},
		// Status-action endpoints are commands: they return the updated sales order without `?include=` expansion. Clients that need expanded sub-resources re-fetch via GET /sales-orders/{id}?include=...
	})
}
