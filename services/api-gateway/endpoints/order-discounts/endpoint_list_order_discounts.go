package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list order discounts.
type ListOrderDiscountsRequest struct {
	apiresource.PaginationRequest
}

type ListOrderDiscountsEndpoint struct{}

func (e *ListOrderDiscountsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListOrderDiscountsRequest, *apiresource.List[apiresource.OrderDiscount]] {
	return &apiendpoint.APIEndpoint[*ListOrderDiscountsRequest, *apiresource.List[apiresource.OrderDiscount]]{
		Title:             "List Order Discounts",
		Description:       "Returns a paginated list of order discounts for the current account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/order-discounts",
		Request:           &ListOrderDiscountsRequest{},
		Response:          &apiresource.List[apiresource.OrderDiscount]{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListOrderDiscountsRequest) (*apiresource.List[apiresource.OrderDiscount], *apierror.APIError) {
			return svc.(OrderDiscountSvc).ListOrderDiscounts
		},
	}
}
