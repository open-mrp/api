package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetSalesOrderRequest is the request to retrieve a single sales order by ID.
type GetSalesOrderRequest struct {
	// The ID of the sales order to retrieve.
	SalesOrderID string `path:"id" validate:"required"`
	// The fields to include in the response.
	Includes []string `query:"include"`
}

type GetSalesOrderEndpoint struct{}

func (e *GetSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSalesOrderRequest, *apiresource.SalesOrderDetail] {
	return &apiendpoint.APIEndpoint[*GetSalesOrderRequest, *apiresource.SalesOrderDetail]{
		Title:             "Get Sales Order",
		Description:       "Returns a single sales order by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/sales-orders/{id}",
		Request:           &GetSalesOrderRequest{},
		Response:          &apiresource.SalesOrderDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSalesOrderRequest) (*apiresource.SalesOrderDetail, *apierror.APIError) {
			return svc.(SalesOrderSvc).GetSalesOrder
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeSalesOrder,
			Fields:     []string{"customer", "bill_to_address", "ship_to_address", "carrier", "service_level", "payment_term", "shipping_term", "order_discount", "lines"},
		}),
	}
}
