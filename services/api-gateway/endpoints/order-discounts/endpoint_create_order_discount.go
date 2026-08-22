package orderdiscountep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/auth-service/pkg/types"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create an order discount.
type CreateOrderDiscountRequest struct {
	// Display name of the discount.
	Name string `json:"name" validate:"required,max=255"`
	// The code a buyer enters to apply this discount to an order.
	//
	// Codes are unique within your account and are compared without regard to letter case, so `SAVE10` collides with `save10`.
	Code string `json:"code" validate:"required,max=255"`
	// The fraction of the order total to take off, as a decimal string.
	//
	// This is a multiplier, not a whole percent: send `0.1` to take 10% off. Only read when `discount_type` is `percentage`. Leaving it out stores `0`, which produces a discount that takes nothing off.
	Percentage field.Optional[string] `json:"percentage,omitzero" format:"decimal"`
	// The flat amount to take off the order total, as a decimal string.
	//
	// Only read when `discount_type` is `amount`. Leaving it out stores `0`, which produces a discount that takes nothing off.
	Amount field.Optional[string] `json:"amount,omitzero" format:"decimal"`
	// How the discount is calculated.
	//
	// - `percentage`: the order total is reduced by the fraction in `percentage`.
	// - `amount`: the order total is reduced by the flat amount in `amount`.
	DiscountType constants.OrderDiscountType `json:"discount_type" validate:"required"`
}

var sampleCreateOrderDiscountRequest = &CreateOrderDiscountRequest{
	Name:         "10% Off",
	Code:         "SAVE10",
	Percentage:   field.Some("0.1"),
	DiscountType: constants.OrderDiscountTypePercentage,
}

func (*CreateOrderDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateOrderDiscountRequest)
}

// Creates an order discount that buyers can then redeem on a sales order by its code.
//
// The code must be unique within your account; reusing a code that another discount already holds returns a conflict error. Creating the discount does not apply it to anything — a discount only affects an order once that order references it.
type CreateOrderDiscountEndpoint struct{}

func (e *CreateOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateOrderDiscountRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*CreateOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Create Order Discount",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/order-discounts",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
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
