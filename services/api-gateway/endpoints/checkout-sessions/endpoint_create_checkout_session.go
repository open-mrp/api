package checkoutsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a customer checkout session.
type CreateCheckoutSessionRequest struct {
	// ID of the sales order to collect payment for.
	//
	// It is recorded on the Stripe session so that the resulting payment is linked back to this order once Stripe reports it as succeeded.
	OrderID string `json:"order_id" validate:"required"`
	// Human-readable order number shown to the customer during checkout.
	//
	// Appears in the name of the single line item on the Stripe payment form, prefixed with `SO #`.
	OrderNumber string `json:"order_number" validate:"required,max=255"`
	// Amount to charge the customer, in cents.
	//
	// Billed in US dollars as one line item covering the whole order.
	OrderTotalCents int64 `json:"order_total_cents" validate:"required"`
	// Customer purchase order (PO) number to associate with the payment.
	//
	// Appears as the description of the checkout line item, prefixed with `PO #`.
	CustomerPO field.Optional[string] `json:"customer_po,omitzero" validate:"omitempty,max=255"`
}

var sampleCreateCheckoutSessionRequest = &CreateCheckoutSessionRequest{
	OrderID:         apiresource.SampleSalesOrderID,
	OrderNumber:     apiresource.SampleSalesOrderNumber,
	OrderTotalCents: 50000,
	CustomerPO:      field.Some("PO-4242"),
}

func (*CreateCheckoutSessionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateCheckoutSessionRequest)
}

// Result of creating a customer checkout session.
type CheckoutSessionResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=checkout_session"`
	// Client secret for the Stripe embedded checkout session.
	//
	// Pass this to Stripe.js to mount the embedded checkout form.
	CheckoutSessionClientSecret string `json:"checkout_session_client_secret" validate:"required,max=255" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleCheckoutSessionResponse = &CheckoutSessionResponse{
	Object:                      constants.ObjectTypeCheckoutSession,
	CheckoutSessionClientSecret: "cs_test_secret_example123",
}

func (*CheckoutSessionResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCheckoutSessionResponse)
}

// Creates an embedded Stripe checkout session for paying a sales order and returns a client secret for use with Stripe.js.
//
// The session is created on the target account's own Stripe integration, so the caller must be a customer user of that account and the account's Stripe integration must be connected and active. On a customer's first checkout, a Stripe customer record is created for them on that integration and reused afterwards. Payment is confirmed asynchronously: Stripe reports the completed payment back through the account's webhook, which links it to the order.
type CreateCheckoutSessionEndpoint struct{}

func (e *CreateCheckoutSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCheckoutSessionRequest, *CheckoutSessionResponse] {
	return (&apiendpoint.APIEndpoint[*CreateCheckoutSessionRequest, *CheckoutSessionResponse]{
		Title:             "Create Customer Checkout Session",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/checkout-sessions",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCheckoutSessionRequest) (*CheckoutSessionResponse, *apierror.APIError) {
			return svc.(CheckoutSessionSvc).CreateCheckoutSession
		},
	})
}
