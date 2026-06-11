package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to delete an order discount.
type DeleteOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
}

// Deletes an order discount and returns the deleted resource.
//
// Deletion is permanent; further requests against the deleted ID return an error.
type DeleteOrderDiscountEndpoint struct{}

func (e *DeleteOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteOrderDiscountRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*DeleteOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Delete Order Discount",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).DeleteOrderDiscount
		},
	})
}
