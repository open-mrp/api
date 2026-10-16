package salesorderep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a sales order by ID.
type RetrieveSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// Retrieves a sales order.
//
// Customer and supplier portal users can only retrieve orders their own account placed.
type RetrieveSalesOrderEndpoint struct{}

func (e *RetrieveSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSalesOrderRequest, *apiresource.SalesOrder] {
	return (&apiendpoint.APIEndpoint[*RetrieveSalesOrderRequest, *apiresource.SalesOrder]{
		Title:             "Retrieve Sales Order",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeSalesOrder,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSalesOrderRequest) (*apiresource.SalesOrder, *apierror.APIError) {
			return svc.(SalesOrderSvc).GetSalesOrder
		},
		IncludeConfig: salesOrderIncludeConfig(),
	})
}
