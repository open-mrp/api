package orderdiscountep

import (
	"context"
	"net/http"

	"github.com/augno/api/services/auth-service/pkg/types"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create an order discount.
type CreateOrderDiscountRequest struct {
	// Display name of the discount.
	Name string `json:"name" validate:"required,max=255"`
	// The code entered to apply this discount to an order.
	//
	// Must be unique within the account.
	Code string `json:"code" validate:"required,max=255"`
	// Percent off as a decimal string (e.g. `10` for 10%).
	//
	// Used when `discount_type` is `percentage`; otherwise `0`.
	Percentage field.Optional[string] `json:"percentage,omitzero" format:"decimal"`
	// Fixed amount off as a decimal string.
	//
	// Used when `discount_type` is `amount`; otherwise `0`.
	Amount field.Optional[string] `json:"amount,omitzero" format:"decimal"`
	// How the discount is calculated.
	//
	// - `percentage`: the discount is a percent off, taken from `percentage`.
	// - `amount`: the discount is a fixed amount off, taken from `amount`.
	DiscountType string `json:"discount_type" validate:"required,max=255"`
}

var sampleCreateOrderDiscountRequest = &CreateOrderDiscountRequest{
	Name:         "10% Off",
	Code:         "SAVE10",
	Percentage:   field.Some("10.000000000000000000000000000000"),
	DiscountType: "percentage",
}

func (*CreateOrderDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateOrderDiscountRequest)
}

// Creates an order discount.
//
// The discount code must be unique within the account; creating a discount with an existing code returns a conflict error.
type CreateOrderDiscountEndpoint struct{}

func (e *CreateOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateOrderDiscountRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*CreateOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Create Order Discount",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/order-discounts",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).CreateOrderDiscount
		},
		LocationFunc: func(resp *apiresource.OrderDiscount) string {
			return "/v1/sales/order-discounts/" + resp.ID
		},
	})
}
