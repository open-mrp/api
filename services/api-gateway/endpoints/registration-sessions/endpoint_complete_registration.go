package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to complete registration for a session.
type CompleteRegistrationRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

// Completes the registration flow by provisioning accounts, roles, and permissions. Requires payment to be confirmed first.
type CompleteRegistrationEndpoint struct{}

func (e *CompleteRegistrationEndpoint) Materialize() *apiendpoint.APIEndpoint[*CompleteRegistrationRequest, *apiresource.CompleteRegistrationResponse] {
	return (&apiendpoint.APIEndpoint[*CompleteRegistrationRequest, *apiresource.CompleteRegistrationResponse]{
		Title:             "Complete Registration",
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
	}).WithDocSource(e)
}
