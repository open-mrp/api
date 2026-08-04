package salesorderstatusep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list sales order statuses.
type ListSalesOrderStatusesRequest struct {
	apiresource.PaginationRequest
}

// Lists the statuses a sales order can be in.
//
// The statuses are platform-provided and the same for every account, so the result is small and stable enough to cache. Use it to label orders in your own interface; an order moves between statuses through its issue, unissue, close, and reopen actions rather than by being assigned a status.
type ListSalesOrderStatusesEndpoint struct{}

func (e *ListSalesOrderStatusesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]] {
	return (&apiendpoint.APIEndpoint[*ListSalesOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]]{
		Title:             "List Sales Order Statuses",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/statuses",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeSalesOrderStatus,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError) {
			return svc.(SalesOrderStatusSvc).ListSalesOrderStatuses
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrderStatus,
			Fields:     []string{"owner"},
		}),
	})
}
