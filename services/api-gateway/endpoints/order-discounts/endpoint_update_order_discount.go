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

// Request to partially update an order discount.
type UpdateOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Discount code.
	Code field.Optional[string] `json:"code,omitzero" validate:"omitempty,max=255"`
	// Percentage value as a decimal string.
	Percentage field.Optional[string] `json:"percentage,omitzero" format:"decimal"`
	// Fixed amount as a decimal string.
	Amount field.Optional[string] `json:"amount,omitzero" format:"decimal"`
	// Discount type: "percentage" or "amount".
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).UpdateOrderDiscount
		},
	})
}
