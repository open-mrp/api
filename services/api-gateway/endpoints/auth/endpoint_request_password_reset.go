package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to request a password reset
type RequestPasswordResetRequest struct {
	// The username or email of the account to reset.
	Identifier string `json:"identifier" validate:"required,identifier"`
	// The account slug, used to redirect the user back to the original account login portal after password reset.
	AccountSlug *string `json:"account_slug,omitempty"`
}

var sampleRequestPasswordResetRequest = &RequestPasswordResetRequest{
	Identifier: apiresource.SampleUserEmail,
}

func (*RequestPasswordResetRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRequestPasswordResetRequest)
}

const requestPasswordResetEndpointDescription string = `This endpoint is used to request a password reset for a user. An email will be sent to the user 
with a link to reset their password.`

type RequestPasswordResetEndpoint struct{}

func (e *RequestPasswordResetEndpoint) Materialize() *apiendpoint.APIEndpoint[*RequestPasswordResetRequest, *apiresource.EmptyResource] {
	return &apiendpoint.APIEndpoint[*RequestPasswordResetRequest, *apiresource.EmptyResource]{
		Title:             "Request Password Reset",
		Description:       requestPasswordResetEndpointDescription,
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
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
