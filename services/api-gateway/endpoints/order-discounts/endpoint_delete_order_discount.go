package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an order discount.
type DeleteOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
}

type DeleteOrderDiscountEndpoint struct{}

func (e *DeleteOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteOrderDiscountRequest, *apiresource.OrderDiscount] {
	return &apiendpoint.APIEndpoint[*DeleteOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Delete Order Discount",
		Description:       "Deletes an order discount by ID.",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		Request:           &DeleteOrderDiscountRequest{},
		Response:          &apiresource.OrderDiscount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).DeleteOrderDiscount
		},
	}
}
