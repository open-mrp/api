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

// Request to close a sales order.
type CloseSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

var sampleCloseSalesOrderRequest = &CloseSalesOrderRequest{}

func (*CloseSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCloseSalesOrderRequest)
}

// Closes a sales order, transitioning it from `issued` to `fulfilled`.
//
// Stamps the order's completion timestamp and closes its pick, packing every pick line that is still open so the pick reads as complete alongside the order. Only an order in `issued` can be closed, and once it is fulfilled it can no longer be deleted, nor can its lines be removed, until it is reopened.
type CloseSalesOrderEndpoint struct{}

func (e *CloseSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CloseSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*CloseSalesOrderRequest, *apiresource.SalesOrder]{
		Title:               "Close Sales Order",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/{id}/actions/close",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CloseSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).CloseSalesOrder
		},
		// Status-action endpoints are commands: they return the updated sales order without `?include=` expansion. Clients that need expanded sub-resources re-fetch via GET /sales-orders/{id}?include=...
	})
}
