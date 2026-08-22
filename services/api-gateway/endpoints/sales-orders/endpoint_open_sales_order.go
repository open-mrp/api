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

// Request to reopen a sales order.
type OpenSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

var sampleOpenSalesOrderRequest = &OpenSalesOrderRequest{}

func (*OpenSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleOpenSalesOrderRequest)
}

// Reopens a sales order, transitioning it from `fulfilled` back to `issued`.
//
// Clears the order's completion timestamp and reopens its pick, unpacking every pick line that is not yet fully picked so the outstanding work can be resumed; lines already picked in full stay packed. Only an order in `fulfilled` can be reopened.
type OpenSalesOrderEndpoint struct{}

func (e *OpenSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*OpenSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*OpenSalesOrderRequest, *apiresource.SalesOrder]{
		Title:               "Reopen Sales Order",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/{id}/actions/open",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *OpenSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).OpenSalesOrder
		},
		// Status-action endpoints are commands: they return the updated sales order without `?include=` expansion. Clients that need expanded sub-resources re-fetch via GET /sales-orders/{id}?include=...
	})
}
