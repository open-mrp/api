package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to partially update an order discount.
type UpdateOrderDiscountRequest struct {
	// Order discount ID.
	OrderDiscountID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Discount code.
	Code *string `json:"code,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Percentage value as a decimal string.
	Percentage *string `json:"percentage,omitempty" nullable:"false" format:"decimal"`
	// Fixed amount as a decimal string.
	Amount *string `json:"amount,omitempty" nullable:"false" format:"decimal"`
	// Discount type: "percentage" or "amount".
	DiscountType *string `json:"discount_type,omitempty" nullable:"false" validate:"omitempty,max=255"`
}

var sampleUpdateOrderDiscountRequest = &UpdateOrderDiscountRequest{
	Name: new("15% Off"),
	Code: new("SAVE15"),
}

func (*UpdateOrderDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateOrderDiscountRequest)
}

type UpdateOrderDiscountEndpoint struct{}

func (e *UpdateOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateOrderDiscountRequest, *apiresource.OrderDiscount] {
	return &apiendpoint.APIEndpoint[*UpdateOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Update Order Discount",
		Description:       "Partially updates an order discount.",
		Method:            http.MethodPatch,
		Route:             "/v1/sales/order-discounts/{id}",
		ContentType:       "application/json",
		Request:           &UpdateOrderDiscountRequest{},
		Response:          &apiresource.OrderDiscount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).UpdateOrderDiscount
		},
	}
}
