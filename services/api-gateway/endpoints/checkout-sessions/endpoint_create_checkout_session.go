package checkoutsessionep

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

// Request to create a customer checkout session.
type CreateCheckoutSessionRequest struct {
	// Sales order ID.
	OrderID string `json:"order_id" validate:"required"`
	// Order number for display.
	OrderNumber string `json:"order_number" validate:"required,max=255"`
	// Order total in cents.
	OrderTotalCents int64 `json:"order_total_cents" validate:"required"`
	// Customer PO number.
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
	// Stripe checkout session client secret for embedded checkout.
	CheckoutSessionClientSecret string `json:"checkout_session_client_secret" validate:"required,max=255" sensitive:"true"` // #nosec G117 - Struct field, not a hardcoded credential
}

var sampleCheckoutSessionResponse = &CheckoutSessionResponse{
	Object:                      constants.ObjectTypeCheckoutSession,
	CheckoutSessionClientSecret: "cs_test_secret_example123",
}

func (*CheckoutSessionResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCheckoutSessionResponse)
}

// Creates an embedded Stripe checkout session for a customer actor and returns a client secret for use with Stripe.js.
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCheckoutSessionRequest) (*CheckoutSessionResponse, *apierror.APIError) {
			return svc.(CheckoutSessionSvc).CreateCheckoutSession
		},
	})
}
