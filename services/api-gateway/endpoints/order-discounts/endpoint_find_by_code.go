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
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
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

// Retrieves an order discount by its code.
//
// Use it to check a code a buyer typed before applying it to an order. Matching ignores letter case. For a buyer (or with `buyer_account_id`), a code the buyer already redeemed on another order, or an archived code, is not found; pass `sales_order_id` to ignore the order being edited.
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
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionRead},
		}},
		ObjectType: constants.ObjectTypeOrderDiscount,
		Extras:     apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *FindOrderDiscountByCodeRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).FindOrderDiscountByCode
		},
	})
}
