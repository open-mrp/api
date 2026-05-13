package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve an order discount.
type RetrieveOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
}

type RetrieveOrderDiscountEndpoint struct{}

func (e *RetrieveOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveOrderDiscountRequest, *apiresource.OrderDiscount] {
	return &apiendpoint.APIEndpoint[*RetrieveOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Retrieve Order Discount",
		Description:       "Returns an order discount by ID.",
		Method:            http.MethodGet,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		Request:           &RetrieveOrderDiscountRequest{},
		Response:          &apiresource.OrderDiscount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).GetOrderDiscount
		},
	}
}
