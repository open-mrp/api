package analyticsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// ListOpenOrderLinesRequest names the sales order whose lines to list.
type ListOpenOrderLinesRequest struct {
	// The sales order's ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// Returns a sales order's sale lines by SKU, each with its unit price, the quantities back-ordered and invoiced in the item's base unit, and its ordered value.
//
// Every sale line of the order is listed, whether or not the order is still open. Sales reps can read only their own orders.
type ListOpenOrderLinesEndpoint struct{}

func (e *ListOpenOrderLinesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOpenOrderLinesRequest, *apiresource.List[apiresource.OpenOrderLine]] {
	return (&apiendpoint.APIEndpoint[*ListOpenOrderLinesRequest, *apiresource.List[apiresource.OpenOrderLine]]{
		Title:               "List Open Order Lines",
		Method:              http.MethodGet,
		Route:               "/v1/core/analytics/open-orders/{id}/lines",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeOpenOrderLine,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOpenOrderLinesRequest) (*apiresource.List[apiresource.OpenOrderLine], *apierror.APIError) {
			return svc.(AnalyticsSvc).ListOpenOrderLines
		},
	})
}
