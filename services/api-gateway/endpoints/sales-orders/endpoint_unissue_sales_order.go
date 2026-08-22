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

// Request to unissue a sales order.
type UnissueSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

var sampleUnissueSalesOrderRequest = &UnissueSalesOrderRequest{}

func (*UnissueSalesOrderRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUnissueSalesOrderRequest)
}

// Unissues a sales order, transitioning it from `issued` back to `estimate`.
//
// Deletes the order's pick, discarding any picking progress recorded against it, and releases the inventory reserved when the order was issued. Only an order in `issued` can be unissued.
type UnissueSalesOrderEndpoint struct{}

func (e *UnissueSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnissueSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*UnissueSalesOrderRequest, *apiresource.SalesOrder]{
		Title:               "Unissue Sales Order",
		Method:              http.MethodPut,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/{id}/actions/unissue",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate}},
		ObjectType:          constants.ObjectTypeSalesOrder,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnissueSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).UnissueSalesOrder
		},
		// Status-action endpoints are commands: they return the updated sales order without `?include=` expansion. Clients that need expanded sub-resources re-fetch via GET /sales-orders/{id}?include=...
	})
}
