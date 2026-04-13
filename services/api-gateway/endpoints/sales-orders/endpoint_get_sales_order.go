package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a sales order by ID.
type GetSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Fields to include in the response.
	Includes []string `query:"include"`
}

type GetSalesOrderEndpoint struct{}

func (e *GetSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSalesOrderRequest, *apiresource.SalesOrderDetail] {
	return &apiendpoint.APIEndpoint[*GetSalesOrderRequest, *apiresource.SalesOrderDetail]{
		Title:             "Get Sales Order",
		Description:       "Returns a sales order by ID.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
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
