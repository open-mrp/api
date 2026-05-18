package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a checkout session for a sales order.
type CheckoutSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Email for the checkout session.
	Email string `json:"email" validate:"required,email"`
	// Redirect URL on success.
	SuccessURL *string `json:"success_url,omitempty"`
	// Redirect URL on cancel.
	CancelURL *string `json:"cancel_url,omitempty"`
}

// Checkout session result.
type CheckoutSalesOrderResponse struct {
	// Checkout URL.
	CheckoutURL string `json:"checkout_url" validate:"required"`
}

func (*CheckoutSalesOrderResponse) SchemaExample() any {
	return map[string]any{"checkout_url": "https://checkout.stripe.com/pay/cs_test_example"}
}

// Creates a checkout session for a sales order.
type CheckoutSalesOrderEndpoint struct{}

func (e *CheckoutSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CheckoutSalesOrderRequest, *CheckoutSalesOrderResponse] {
	return (&apiendpoint.APIEndpoint[*CheckoutSalesOrderRequest, *CheckoutSalesOrderResponse]{
		Title:             "Checkout Sales Order",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/checkout",
		Request:           &CheckoutSalesOrderRequest{},
		Response:          &CheckoutSalesOrderResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError) {
			return svc.(SalesOrderSvc).CheckoutSalesOrder
		},
	}).WithDocSource(e)
}
