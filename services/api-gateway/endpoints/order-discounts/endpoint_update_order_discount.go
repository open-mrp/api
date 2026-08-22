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

// Request to partially update an order discount.
type UpdateOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
	// Display name of the discount.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// The code a buyer enters to apply this discount to an order.
	//
	// Codes are unique within your account and are compared without regard to letter case.
	Code field.Optional[string] `json:"code,omitzero" validate:"omitempty,max=255"`
	// The fraction of the order total to take off, as a decimal string.
	//
	// This is a multiplier, not a whole percent: send `0.1` to take 10% off. Only read when `discount_type` is `percentage`.
	Percentage field.Optional[string] `json:"percentage,omitzero" format:"decimal"`
	// The flat amount to take off the order total, as a decimal string.
	//
	// Only read when `discount_type` is `amount`.
	Amount field.Optional[string] `json:"amount,omitzero" format:"decimal"`
	// How the discount is calculated.
	//
	// - `percentage`: the order total is reduced by the fraction in `percentage`.
	// - `amount`: the order total is reduced by the flat amount in `amount`.
	//
	// Switching the type does not move the stored figure across, so send the matching `percentage` or `amount` in the same request or the discount will take nothing off.
	DiscountType field.Optional[constants.OrderDiscountType] `json:"discount_type,omitzero"`
}

var sampleUpdateOrderDiscountRequest = &UpdateOrderDiscountRequest{
	Name: field.Some("15% Off"),
	Code: field.Some("SAVE15"),
}

func (*UpdateOrderDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateOrderDiscountRequest)
}

// Partially updates an order discount.
//
// Only the fields you send are changed; the rest keep their current values. Changing `code` to one another discount already holds returns a conflict error. Edits apply to future orders only — orders that already used this discount keep the reduction they were given.
type UpdateOrderDiscountEndpoint struct{}

func (e *UpdateOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateOrderDiscountRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*UpdateOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Update Order Discount",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeOrderDiscount,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainDiscounts, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).UpdateOrderDiscount
		},
	})
}
