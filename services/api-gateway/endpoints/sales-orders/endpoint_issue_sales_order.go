package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to issue a sales order.
type IssueSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Whether to notify the customer.
	//
	// When `true`, the order acknowledgement email is sent to the contacts configured on the order and the order's `acknowledgment_status` is set to `sent`.
	NotifyCustomer bool `json:"notify_customer"`
}

var sampleIssueSalesOrderRequest = &IssueSalesOrderRequest{NotifyCustomer: true}

func (*IssueSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleIssueSalesOrderRequest)
}

// Issues a sales order, transitioning it from `estimate` to `issued`.
//
// Issuing commits the order for fulfillment: a pick is created for the order's sale lines and inventory is reserved for each line tied to an inventory item.
type IssueSalesOrderEndpoint struct{}

func (e *IssueSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*IssueSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*IssueSalesOrderRequest, *apiresource.SalesOrder]{
		Title:               "Issue Sales Order",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/{id}/actions/issue",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *IssueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).IssueSalesOrder
		},
		// Status-action endpoints are commands: they return the updated sales
		// order without `?include=` expansion. Clients that need expanded
		// sub-resources re-fetch via GET /sales-orders/{id}?include=...
	})
}
