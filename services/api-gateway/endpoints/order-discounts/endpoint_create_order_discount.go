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

// Request to create an order discount.
type CreateOrderDiscountRequest struct {
	// Display name.
	Name string `json:"name" validate:"required,max=255"`
	// Discount code.
	Code string `json:"code" validate:"required,max=255"`
	// Percentage value as a decimal string. Required when discount_type is "percentage".
	Percentage field.Optional[string] `json:"percentage,omitzero" format:"decimal"`
	// Fixed amount as a decimal string. Required when discount_type is "amount".
	Amount field.Optional[string] `json:"amount,omitzero" format:"decimal"`
	// Discount type: "percentage" or "amount".
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateOrderDiscountRequest) (*apiresource.OrderDiscount, *apierror.APIError) {
			return svc.(OrderDiscountSvc).CreateOrderDiscount
		},
		LocationFunc: func(resp *apiresource.OrderDiscount) string {
			return "/v1/sales/order-discounts/" + resp.ID
		},
	})
}
