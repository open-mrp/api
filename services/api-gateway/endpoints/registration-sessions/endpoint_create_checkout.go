package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to create a checkout session for a registration session
type CreateCheckoutRequest struct {
	// The session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

const createCheckoutEndpointDescription string = `Creates a Stripe checkout session for a registration session. Creates the Stripe customer,
product, price, and checkout session. Uses recovery points for safe retries after failures.`

type CreateCheckoutEndpoint struct{}

func (e *CreateCheckoutEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateCheckoutRequest, *apiresource.CreateCheckoutResponse] {
	return &apiendpoint.APIEndpoint[*CreateCheckoutRequest, *apiresource.CreateCheckoutResponse]{
		Title:             "Create Registration Checkout",
		Description:       createCheckoutEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/actions/checkout",
		ContentType:       "application/json",
		Request:           &CreateCheckoutRequest{},
		Response:          &apiresource.CreateCheckoutResponse{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateCheckoutRequest) (*apiresource.CreateCheckoutResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).CreateCheckout
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
