package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to find an order discount by code.
type FindOrderDiscountByCodeRequest struct {
	// The discount code to look up, as the buyer typed it.
	//
	// Matching ignores letter case, so `save10` finds a discount stored as `SAVE10`.
	Code string `json:"code" validate:"required"`
	// The buyer account to check for prior use of this code.
	//
	// When set, the lookup returns a not-found error if that buyer has already redeemed the discount on another order, so a one-use-per-customer code can be rejected before it is attached to a new one. Customer callers cannot set this — their own account is always used.
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

// Validates a discount code and returns the matching order discount, so a code a buyer typed can be attached to an order.
//
// When `buyer_account_id` is provided, or the caller is a customer user, the lookup also verifies that the buyer has not already redeemed the discount on another order, and reports an already-redeemed code as not found. Pass `sales_order_id` to exclude an order the buyer is currently editing from that check.
type FindOrderDiscountByCodeEndpoint struct{}

func (e *FindOrderDiscountByCodeEndpoint) Materialize() *apiendpoint.APIEndpoint[*FindOrderDiscountByCodeRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*FindOrderDiscountByCodeRequest, *apiresource.OrderDiscount]{
		Title:             "Find Order Discount by Code",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/order-discounts/actions/find-by-code",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead},
		},
		ObjectType: constants.ObjectTypeOrderDiscount,
		Extras:     apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *FindOrderDiscountByCodeRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).FindOrderDiscountByCode
		},
	})
}
