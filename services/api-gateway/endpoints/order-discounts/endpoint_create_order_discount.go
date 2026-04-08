package orderdiscountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateOrderDiscountRequest is the request to create a new order discount.
type CreateOrderDiscountRequest struct {
	// The display name of the discount.
	Name string `json:"name" validate:"required,max=255"`
	// The unique code for this discount.
	Code string `json:"code" validate:"required,max=255"`
	// The percentage value of the discount as a decimal string. Required when discount_type is "percentage".
	Percentage *string `json:"percentage,omitempty" format:"decimal"`
	// The fixed amount of the discount as a decimal string. Required when discount_type is "amount".
	Amount *string `json:"amount,omitempty" format:"decimal"`
	// The type of discount: "percentage" or "amount".
	DiscountType string `json:"discount_type" validate:"required,max=255"`
}

var sampleCreateOrderDiscountRequest = &CreateOrderDiscountRequest{
	Name:         "10% Off",
	Code:         "SAVE10",
	Percentage:   new("10.000000000000000000000000000000"),
	DiscountType: "percentage",
}

func (*CreateOrderDiscountRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateOrderDiscountRequest)
}

type CreateOrderDiscountEndpoint struct{}

func (e *CreateOrderDiscountEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateOrderDiscountRequest, *apiresource.OrderDiscount] {
	return &apiendpoint.APIEndpoint[*CreateOrderDiscountRequest, *apiresource.OrderDiscount]{
		Title:             "Create Order Discount",
		Description:       "Creates a new order discount.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/order-discounts",
		Request:           &CreateOrderDiscountRequest{},
		Response:          &apiresource.OrderDiscount{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).CreateOrderDiscount
		},
	}
}
