package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// FindOrderDiscountByCodeRequest is the request to find an order discount by code.
type FindOrderDiscountByCodeRequest struct {
	// The discount code to look up.
	Code string `json:"code" validate:"required"`
	// Optional buyer account ID to scope the lookup.
	BuyerAccountID *string `json:"buyer_account_id,omitempty"`
	// Optional sales order ID to scope the lookup.
	SalesOrderID *string `json:"sales_order_id,omitempty"`
}

var sampleFindOrderDiscountByCodeRequest = &FindOrderDiscountByCodeRequest{
	Code: "SAVE10",
}

func (*FindOrderDiscountByCodeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleFindOrderDiscountByCodeRequest)
}

type FindOrderDiscountByCodeEndpoint struct{}

func (e *FindOrderDiscountByCodeEndpoint) Materialize() *apiendpoint.APIEndpoint[*FindOrderDiscountByCodeRequest, *apiresource.OrderDiscount] {
	return &apiendpoint.APIEndpoint[*FindOrderDiscountByCodeRequest, *apiresource.OrderDiscount]{
		Title:             "Find Order Discount by Code",
		Description:       "Finds an order discount by its unique code, optionally scoped to a buyer account or sales order.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/order-discounts/actions/find-by-code",
		Request:           &FindOrderDiscountByCodeRequest{},
		Response:          &apiresource.OrderDiscount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *FindOrderDiscountByCodeRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).FindOrderDiscountByCode
		},
	}
}
