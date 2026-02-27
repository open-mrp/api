package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// The request to complete registration for a registration session
type CompleteRegistrationRequest struct {
	// The session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

const completeRegistrationEndpointDescription string = `Completes the registration flow by creating the production account, sandbox account,
roles, permissions, and account-user records. Requires payment to be completed first.`

type CompleteRegistrationEndpoint struct{}

func (e *CompleteRegistrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CompleteRegistrationRequest, *apiresource.CompleteRegistrationResponse] {
	return &apiendpoint.APIEndpoint[*CompleteRegistrationRequest, *apiresource.CompleteRegistrationResponse]{
		Title:             "Complete Registration",
		Description:       completeRegistrationEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/accounts",
		ContentType:       "application/json",
		Request:           &CompleteRegistrationRequest{},
		Response:          &apiresource.CompleteRegistrationResponse{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CompleteRegistrationRequest) (*apiresource.CompleteRegistrationResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).CompleteRegistration
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
