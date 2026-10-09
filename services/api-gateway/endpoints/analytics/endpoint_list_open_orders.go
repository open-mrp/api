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

// ListOpenOrdersRequest selects the open sales orders to list.
type ListOpenOrdersRequest struct {
	OpenOrderFilters
	// Opaque cursor from a previous page's `next_page_url` or `previous_page_url`. Omit for the first page.
	Cursor *string `query:"cursor"`
	// Maximum number of orders to return.
	Limit int32 `query:"limit" default:"10" validate:"min=1,max=100"`
}

// Returns the open sales orders, most recently issued first, each with the number and ordered value of its sale lines the filters count.
//
// An open order is a sales order that has been issued and not yet completed. The product-line and item filters choose which lines count; an order with no counted line is left out. Sales reps see only their own orders.
type ListOpenOrdersEndpoint struct{}

func (e *ListOpenOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOpenOrdersRequest, *apiresource.List[apiresource.OpenOrder]] {
	return (&apiendpoint.APIEndpoint[*ListOpenOrdersRequest, *apiresource.List[apiresource.OpenOrder]]{
		Title:               "List Open Orders",
		Method:              http.MethodPut,
		Route:               "/v1/core/analytics/open-orders",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		AgentTool:           true,
		Preview:             true,
		ReadOnly:            true,
		ObjectType:          constants.ObjectTypeOpenOrder,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainInvoices, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOpenOrdersRequest) (*apiresource.List[apiresource.OpenOrder], *apierror.APIError) {
			return svc.(AnalyticsSvc).ListOpenOrders
		},
	})
}
