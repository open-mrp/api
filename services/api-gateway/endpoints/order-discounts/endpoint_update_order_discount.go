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

// Request to partially update an order discount.
type UpdateOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
	// Display name of the discount.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// The code entered to apply this discount to an order.
	//
	// Must be unique within the account.
	Code field.Optional[string] `json:"code,omitzero" validate:"omitempty,max=255"`
	// Percent off as a decimal string (e.g. `10` for 10%).
	//
	// Used when `discount_type` is `percentage`.
	Percentage field.Optional[string] `json:"percentage,omitzero" format:"decimal"`
	// Fixed amount off as a decimal string.
	//
	// Used when `discount_type` is `amount`.
	Amount field.Optional[string] `json:"amount,omitzero" format:"decimal"`
	// How the discount is calculated.
	//
	// - `percentage`: the discount is a percent off, taken from `percentage`.
	// - `amount`: the discount is a fixed amount off, taken from `amount`.
	DiscountType field.Optional[string] `json:"discount_type,omitzero" validate:"omitempty,max=255"`
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
// Only the provided fields are changed. Changing `code` to one already used by another discount returns a conflict error.
type UpdateOrderDiscountEndpoint struct{}

func (e *UpdateOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateOrderDiscountRequest, *apiresource.OrderDiscount] {
	return (&apiendpoint.APIEndpoint[*UpdateOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Update Order Discount",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
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
