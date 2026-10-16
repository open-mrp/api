package salesorderstatusep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list sales order statuses.
type ListSalesOrderStatusesRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
}

// Lists the statuses a sales order can be in.
//
// Kept for 1.0.forge-preview.9 callers only; on 1.0.forge a sales order's `status` is a fixed set of values.
type ListSalesOrderStatusesEndpoint struct{}

func (e *ListSalesOrderStatusesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListSalesOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]] {
	return (&apiendpoint.APIEndpoint[*ListSalesOrderStatusesRequest, *apiresource.List[apiresource.SalesOrderStatus]]{
		Title:             "List Sales Order Statuses",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/statuses",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
