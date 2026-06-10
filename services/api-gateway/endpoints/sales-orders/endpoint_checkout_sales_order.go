package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a checkout session for a sales order.
type CheckoutSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Email for the checkout session.
	Email string `json:"email" validate:"required,email"`
	// Redirect URL on success.
	SuccessURL field.Optional[string] `json:"success_url,omitzero"`
	// Redirect URL on cancel.
	CancelURL field.Optional[string] `json:"cancel_url,omitzero"`
}

// Checkout session result.
type CheckoutSalesOrderResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=checkout_sales_order_response"`
	// Checkout URL.
	CheckoutURL string `json:"checkout_url" validate:"required"`
}

var sampleCheckoutSalesOrderResponse = &CheckoutSalesOrderResponse{
	Object:      constants.ObjectTypeCheckoutSalesOrderResponse,
	CheckoutURL: "https://checkout.stripe.com/pay/cs_test_example",
}

func (*CheckoutSalesOrderResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCheckoutSalesOrderResponse)
}

// Creates a checkout session for a sales order.
type CheckoutSalesOrderEndpoint struct{}

func (e *CheckoutSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CheckoutSalesOrderRequest, *CheckoutSalesOrderResponse] {
	return (&apiendpoint.APIEndpoint[*CheckoutSalesOrderRequest, *CheckoutSalesOrderResponse]{
		Title:             "Checkout Sales Order",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/checkout",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError) {
			return svc.(SalesOrderSvc).CheckoutSalesOrder
		},
	})
}
