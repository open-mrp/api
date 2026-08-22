package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to complete registration for a session.
type CompleteRegistrationRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

// Completes a registration session by creating the account the registrant signed up for.
//
// The registering user becomes an administrator of the new account, and a paired sandbox account is provisioned alongside it. Requires a user to have been created for the session and an account name to have been supplied; paid plans additionally require confirmed payment, and their subscription starts here. If the selected plan has reached its signup capacity the request fails and the registration is added to a waiting list. Returns the ID of the new account.
type CompleteRegistrationEndpoint struct{}

func (e *CompleteRegistrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CompleteRegistrationRequest, *apiresource.CompleteRegistrationResponse] {
	return (&apiendpoint.APIEndpoint[*CompleteRegistrationRequest, *apiresource.CompleteRegistrationResponse]{
		Title:             "Complete Registration",
		Method:            http.MethodPost,
		Route:             "/v1/auth/registration-sessions/{session_id}/accounts",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CompleteRegistrationRequest) (*apiresource.CompleteRegistrationResponse, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).CompleteRegistration
		},
	})
}
