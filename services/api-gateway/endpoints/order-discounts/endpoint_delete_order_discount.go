package orderdiscountep

import (
	"context"
	"net/http"

	"github.com/augno/api/services/auth-service/pkg/types"

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

// Deletes an order discount and returns it as it was just before deletion.
//
// Deletion is permanent; further requests against the deleted ID return an error.
//
// The code can no longer be redeemed, but sales orders that already used the discount keep the reduction that was applied to them; their totals are not recalculated.
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
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionDelete},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).DeleteOrderDiscount
		},
	})
}
