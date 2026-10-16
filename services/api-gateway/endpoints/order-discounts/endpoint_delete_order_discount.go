package orderdiscountep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to delete an order discount.
type DeleteOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
}

// Deletes an order discount.
//
// The code can no longer be applied; orders that already used it keep their discount. Archive it instead by setting `status` to `archived` to keep its history.
type DeleteOrderDiscountEndpoint struct{}

func (e *DeleteOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeleteOrderDiscountRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeleteOrderDiscountRequest, *object.Deleted]{
		Title:             "Delete Order Discount",
		Method:            http.MethodDelete,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionDelete},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeleteOrderDiscountRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(OrderDiscountSvc).DeleteOrderDiscount, constants.ObjectTypeOrderDiscount, "id")
		},
	})
}
