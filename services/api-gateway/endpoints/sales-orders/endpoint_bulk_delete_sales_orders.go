package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to bulk delete sales orders.
type BulkDeleteSalesOrdersRequest struct {
	// IDs of the sales orders to delete.
	SalesOrderIDs []string `json:"sales_order_ids" validate:"required"`
}

var sampleBulkDeleteSalesOrdersRequest = &BulkDeleteSalesOrdersRequest{
	SalesOrderIDs: []string{apiresource.SampleSalesOrderID},
}

func (*BulkDeleteSalesOrdersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkDeleteSalesOrdersRequest)
}

// Deletes multiple sales orders in a single atomic operation.
//
// Fulfilled orders cannot be deleted; if any requested order fails this check, no orders are deleted.
type BulkDeleteSalesOrdersEndpoint struct{}

func (e *BulkDeleteSalesOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkDeleteSalesOrdersRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*BulkDeleteSalesOrdersRequest, *apiresource.EmptyResource]{
		Title:               "Bulk Delete Sales Orders",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/actions/bulk-delete",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkDeleteSalesOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SalesOrderSvc).BulkDeleteSalesOrders
		},
	})
}
