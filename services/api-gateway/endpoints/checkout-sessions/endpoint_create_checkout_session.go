package checkoutsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// CreateCheckoutSessionRequest is the request to create a customer checkout session.
type CreateCheckoutSessionRequest struct {
	// The ID of the sales order.
	OrderID string `json:"order_id" validate:"required"`
	// The order number for display.
	OrderNumber string `json:"order_number" validate:"required"`
	// The order total in cents.
	OrderTotalCents int64 `json:"order_total_cents" validate:"required"`
	// The customer PO number, if any.
	CustomerPO *string `json:"customer_po,omitempty"`
}

// CheckoutSessionResponse represents the result of creating a customer checkout session.
type CheckoutSessionResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=checkout_session"`
	// The Stripe checkout session client secret for embedded checkout.
	CheckoutSessionClientSecret string `json:"checkout_session_client_secret" validate:"required"`
}

func (*CheckoutSessionResponse) SchemaExample() any {
	return map[string]any{
		"object":                         "checkout_session",
		"checkout_session_client_secret": "cs_test_secret_example123",
	}
}

type CreateCheckoutSessionEndpoint struct{}

func (e *CreateCheckoutSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCheckoutSessionRequest, *CheckoutSessionResponse] {
	return &apiendpoint.APIEndpoint[*CreateCheckoutSessionRequest, *CheckoutSessionResponse]{
		Title:             "Create Customer Checkout Session",
		Description:       "Creates an embedded Stripe checkout session for a customer actor and returns a client secret for use with Stripe.js.",
		Method:            http.MethodPost,
		Route:             "/v1/sales/checkout-sessions",
		Request:           &CreateCheckoutSessionRequest{},
		Response:          &CheckoutSessionResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCheckoutSessionRequest) (*CheckoutSessionResponse, *apierror.APIError) {
			return svc.(CheckoutSessionSvc).CreateCheckoutSession
		},
	}
}
