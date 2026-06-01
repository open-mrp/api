package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list order discounts.
type ListOrderDiscountsRequest struct {
	apiresource.PaginationRequest
}

// Returns a paginated list of order discounts for the current account.
type ListOrderDiscountsEndpoint struct{}

func (e *ListOrderDiscountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOrderDiscountsRequest, *apiresource.List[apiresource.OrderDiscount]] {
	return (&apiendpoint.APIEndpoint[*ListOrderDiscountsRequest, *apiresource.List[apiresource.OrderDiscount]]{
		Title:             "List Order Discounts",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/order-discounts",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOrderDiscountsRequest) (*apiresource.List[apiresource.OrderDiscount], *apierror.APIError) {
			return svc.(OrderDiscountSvc).ListOrderDiscounts
		},
	})
}
