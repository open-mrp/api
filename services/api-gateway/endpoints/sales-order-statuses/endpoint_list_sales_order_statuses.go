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

type ListSalesOrderStatusesEndpoint struct{}

func (e *ListSalesOrderStatusesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]] {
	return &apiendpoint.APIEndpoint[*ListSalesOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]]{
		Title:             "List Sales Order Statuses",
		Description:       "Returns a paginated list of sales order statuses.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/statuses",
		Request:           &ListSalesOrderStatusesRequest{},
		Response:          &apiresource.List[apiresource.SalesOrderStatus]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListSalesOrderStatusesRequest) (*apiresource.List[apiresource.SalesOrderStatus], *apierror.APIError) {
			return svc.(SalesOrderStatusSvc).ListSalesOrderStatuses
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrderStatus,
			Fields:     []string{"owner", "owner.account"},
		}),
	}
}
