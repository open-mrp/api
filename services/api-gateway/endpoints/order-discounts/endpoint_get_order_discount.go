package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// GetOrderDiscountRequest is the request to retrieve a single order discount.
type GetOrderDiscountRequest struct {
	// The ID of the order discount to retrieve.
	OrderDiscountID string `path:"id" validate:"required"`
}

type GetOrderDiscountEndpoint struct{}

func (e *GetOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetOrderDiscountRequest, *apiresource.OrderDiscount] {
	return &apiendpoint.APIEndpoint[*GetOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Get Order Discount",
		Description:       "Returns a single order discount by its ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		Request:           &GetOrderDiscountRequest{},
		Response:          &apiresource.OrderDiscount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).GetOrderDiscount
		},
	}
}
