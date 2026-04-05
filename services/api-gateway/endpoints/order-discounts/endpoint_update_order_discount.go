package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateOrderDiscountRequest is the request to partially update an order discount.
type UpdateOrderDiscountRequest struct {
	// The ID of the order discount to update.
	OrderDiscountID string `path:"id" validate:"required"`
	// The display name of the discount.
	Name *string `json:"name,omitempty"`
	// The unique code for this discount.
	Code *string `json:"code,omitempty"`
	// The percentage value of the discount as a decimal string.
	Percentage *string `json:"percentage,omitempty" format:"decimal"`
	// The fixed amount of the discount as a decimal string.
	Amount *string `json:"amount,omitempty" format:"decimal"`
	// The type of discount: "percentage" or "amount".
	DiscountType *string `json:"discount_type,omitempty"`
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
