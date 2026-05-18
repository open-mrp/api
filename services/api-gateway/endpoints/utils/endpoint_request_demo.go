package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to submit a demo request.
type RequestDemoRequest struct {
	// Name of the requester.
	Name string `json:"name" validate:"required"`
	// Email address of the requester.
	Email string `json:"email" validate:"required,custom_email"`
	// Company name.
	Company string `json:"company" validate:"required"`
	// Phone number.
	PhoneNumber *string `json:"phone_number"`
	// Message from the requester.
	Message *string `json:"message"`
}

var exampleRequestDemoRequest = &RequestDemoRequest{
	Name:    "Jane Smith",
	Email:   "jane@example.com",
	Company: "Acme Corp",
}

func (*RequestDemoRequest) SchemaExample() any {
	return exampleRequestDemoRequest
}

// Submits a demo request from a prospective customer.
type RequestDemoEndpoint struct{}

func (e *RequestDemoEndpoint) Materialize() *apiendpoint.APIEndpoint[*RequestDemoRequest, *apiresource.MessageResource] {
	return (&apiendpoint.APIEndpoint[*RequestDemoRequest, *apiresource.MessageResource]{
		Title:             "Request Demo",
		Method:            http.MethodPost,
		Route:             "/v1/core/actions/request-demo",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RequestDemoRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(UtilsSvc).RequestDemo
		},
	})
}
