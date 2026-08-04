package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a checkout session for a sales order.
type CheckoutSalesOrderRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
	// Email address to send the checkout link to.
	Email string `json:"email" validate:"required,email"`
}

// Checkout session result.
type CheckoutSalesOrderResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=checkout_sales_order"`
	// URL of the hosted payment page where the customer completes the checkout.
	CheckoutURL string `json:"checkout_url" validate:"required"`
}

var sampleCheckoutSalesOrderResponse = &CheckoutSalesOrderResponse{
	Object:      constants.ObjectTypeCheckoutSalesOrderResponse,
	CheckoutURL: "https://checkout.stripe.com/pay/cs_test_example",
}

func (*CheckoutSalesOrderResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCheckoutSalesOrderResponse)
}

// Creates a hosted payment checkout session for a sales order.
//
// Requires an active Stripe integration on the account and a customer that already exists in Stripe. The customer is charged a single amount covering every line on the order, including its freight and discount lines, and the checkout link is emailed to the address provided. Fails with a conflict if the order already has a payment.
type CheckoutSalesOrderEndpoint struct{}

func (e *CheckoutSalesOrderEndpoint) Materialize() *apiendpoint.APIEndpoint[*CheckoutSalesOrderRequest, *CheckoutSalesOrderResponse] {
	return (&apiendpoint.APIEndpoint[*CheckoutSalesOrderRequest, *CheckoutSalesOrderResponse]{
		Title:               "Checkout Sales Order",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/{id}/checkout",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSalesOrders, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CheckoutSalesOrderRequest) (*CheckoutSalesOrderResponse, *apierror.APIError) {
			return svc.(SalesOrderSvc).CheckoutSalesOrder
		},
	})
}
