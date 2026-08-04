package authep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request for a password reset.
type RequestPasswordResetRequest struct {
	// Username or email of the user whose password should be reset.
	//
	// The reset link is always sent to the email address on file for the matched user, and no email is sent if that user has no address on file.
	Identifier string `json:"identifier" validate:"required,identifier"`
	// Slug of the customer portal the request came from.
	//
	// Scopes the emailed reset link to that portal so the user sets their new password there instead of on the generic dashboard. Accounts with a verified custom portal domain use that domain in the link instead of the slug.
	AccountSlug field.Optional[string] `json:"account_slug,omitzero"`
}

var sampleRequestPasswordResetRequest = &RequestPasswordResetRequest{
	Identifier: apiresource.SampleUserEmail,
}

func (*RequestPasswordResetRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleRequestPasswordResetRequest)
}

// Sends a password reset email to the user.
//
// Always returns an accepted response, whether or not the identifier matches a known user, so it does not reveal which identifiers exist. Reset links expire 15 minutes after they are issued.
type RequestPasswordResetEndpoint struct{}

func (e *RequestPasswordResetEndpoint) Materialize() *apiendpoint.APIEndpoint[*RequestPasswordResetRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*RequestPasswordResetRequest, *apiresource.EmptyResource]{
		Title:             "Request Password Reset",
		Method:            http.MethodPost,
		Route:             "/v1/auth/passwords/actions/request-reset",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RequestPasswordResetRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(AuthSvc).RequestPasswordReset
		},
	})
}
