package utilsep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to be contacted for a product demo.
type RequestDemoRequest struct {
	// Full name of the person requesting the demo.
	Name string `json:"name" validate:"required"`
	// Email address to reach the requester at.
	Email string `json:"email" validate:"required,custom_email"`
	// Name of the company the requester represents.
	Company string `json:"company" validate:"required"`
	// Phone number to reach the requester at.
	PhoneNumber field.Optional[string] `json:"phone_number,omitzero"`
	// Free-form note from the requester about what they would like to see.
	Message field.Optional[string] `json:"message,omitzero"`
}

var sampleRequestDemoRequest = &RequestDemoRequest{
	Name:    "Jane Smith",
	Email:   "jane@example.com",
	Company: "Acme Corp",
}

func (*RequestDemoRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRequestDemoRequest)
}

// Submits a demo request from a prospective customer for the OpenMRP team to follow up on.
//
// The request creates no account, user, or other resource, and there is no endpoint to read it back. The response carries a confirmation message suitable for display.
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
