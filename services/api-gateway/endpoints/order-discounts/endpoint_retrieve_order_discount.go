package orderdiscountep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/auth-service/pkg/types"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to retrieve an order discount.
type RetrieveOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
}

// Returns an order discount by ID.
type RetrieveOrderDiscountEndpoint struct{}

func (e *RetrieveOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveOrderDiscountRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*RetrieveOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Retrieve Order Discount",
		Method:            http.MethodGet,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).GetOrderDiscount
		},
	})
}
