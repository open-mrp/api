package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to find an order discount by code.
type FindOrderDiscountByCodeRequest struct {
	// The discount code to look up.
	Code string `json:"code" validate:"required"`
	// Buyer account ID to check for prior usage.
	//
	// When set, the lookup returns a not-found error if this buyer has already used the discount on another order.
	BuyerAccountID field.Optional[string] `json:"buyer_account_id,omitzero"`
	// Sales order ID to exclude from the prior-usage check.
	//
	// Set this when re-validating a code on an existing order so the order's own usage does not count against the buyer.
	SalesOrderID field.Optional[string] `json:"sales_order_id,omitzero"`
}

var sampleFindOrderDiscountByCodeRequest = &FindOrderDiscountByCodeRequest{
	Code: "SAVE10",
}

func (*FindOrderDiscountByCodeRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleFindOrderDiscountByCodeRequest)
}

// Looks up an order discount by its code.
//
// When `buyer_account_id` is provided (or the caller is a customer user), the lookup also verifies the buyer has not already used the discount on another order, returning a not-found error if they have. Pass `sales_order_id` to exclude an existing order from that check.
type FindOrderDiscountByCodeEndpoint struct{}

func (e *FindOrderDiscountByCodeEndpoint) Materialize() *apiendpoint.APIEndpoint[*FindOrderDiscountByCodeRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*FindOrderDiscountByCodeRequest, *apiresource.OrderDiscount]{
		Title:             "Find Order Discount by Code",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/order-discounts/actions/find-by-code",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *FindOrderDiscountByCodeRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).FindOrderDiscountByCode
		},
	})
}
