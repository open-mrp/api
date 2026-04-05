package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// RequestDemoRequest is the request to submit a demo request.
type RequestDemoRequest struct {
	// The name of the person requesting the demo.
	Name string `json:"name" validate:"required"`
	// The email address of the person requesting the demo.
	Email string `json:"email" validate:"required,custom_email"`
	// The company name of the person requesting the demo.
	Company string `json:"company" validate:"required"`
	// The phone number of the person requesting the demo.
	PhoneNumber *string `json:"phone_number"`
	// An optional message from the person requesting the demo.
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

type RequestDemoEndpoint struct{}

func (e *RequestDemoEndpoint) Materialize() *apiendpoint.APIEndpoint[*RequestDemoRequest, *apiresource.MessageResource] {
	return &apiendpoint.APIEndpoint[*RequestDemoRequest, *apiresource.MessageResource]{
		Title:             "Request Demo",
		Description:       "Submits a demo request from a prospective customer.",
		Method:            http.MethodPost,
		Route:             "/v1/core/actions/request-demo",
		ContentType:       "application/json",
		Request:           &RequestDemoRequest{},
		Response:          &apiresource.MessageResource{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RequestDemoRequest) (*apiresource.MessageResource, *apierror.APIError) {
			return svc.(UtilsSvc).RequestDemo
		},
	}
}
