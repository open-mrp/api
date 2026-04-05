package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// BulkDeleteSalesOrdersRequest is the request to delete multiple sales orders.
type BulkDeleteSalesOrdersRequest struct {
	// The IDs of the sales orders to delete.
	SalesOrderIDs []string `json:"sales_order_ids" validate:"required"`
}

var sampleBulkDeleteSalesOrdersRequest = &BulkDeleteSalesOrdersRequest{
	SalesOrderIDs: []string{apiresource.SampleSalesOrderDetailID},
}

func (*BulkDeleteSalesOrdersRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkDeleteSalesOrdersRequest)
}

type BulkDeleteSalesOrdersEndpoint struct{}

func (e *BulkDeleteSalesOrdersEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkDeleteSalesOrdersRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*BulkDeleteSalesOrdersRequest, *apiresource.EmptyResource]{
		Title:             "Bulk Delete Sales Orders",
		Description:       "Deletes multiple sales orders in a single operation.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/sales-orders/actions/bulk-delete",
		Request:           &BulkDeleteSalesOrdersRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkDeleteSalesOrdersRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(SalesOrderSvc).BulkDeleteSalesOrders
		},
	}
}
