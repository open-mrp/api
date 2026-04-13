package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request for a password reset.
type RequestPasswordResetRequest struct {
	// Username or email of the account to reset.
	Identifier string `json:"identifier" validate:"required,identifier"`
	// Account slug for redirecting to the original login portal after password reset.
	AccountSlug *string `json:"account_slug,omitempty"`
}

var sampleRequestPasswordResetRequest = &RequestPasswordResetRequest{
	Identifier: apiresource.SampleUserEmail,
}

func (*RequestPasswordResetRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRequestPasswordResetRequest)
}

type RequestPasswordResetEndpoint struct{}

func (e *RequestPasswordResetEndpoint) Materialize() *apiendpoint.APIEndpoint[*RequestPasswordResetRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RequestPasswordResetRequest, *apiresource.EmptyResource]{
		Title:             "Request Password Reset",
		Description:       "Sends a password reset email to the user.",
		Method:            http.MethodPost,
		Route:             "/v1/auth/passwords/actions/request-reset",
		ContentType:       "application/json",
		Request:           &RequestPasswordResetRequest{},
		Response:          &apiresource.EmptyResource{},
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RequestPasswordResetRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).RequestPasswordReset
		},
	}
}
