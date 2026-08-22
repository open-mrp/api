package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to get a registration session.
type RetrieveSessionRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

// Returns a registration session by ID, including its current step and associated user and account details.
type RetrieveSessionEndpoint struct{}

func (e *RetrieveSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveSessionRequest, *apiresource.RegistrationSession] {
	return (&apiendpoint.APIEndpoint[*RetrieveSessionRequest, *apiresource.RegistrationSession]{
		Title:             "Retrieve Registration Session",
		Method:            http.MethodGet,
		Route:             "/v1/auth/registration-sessions/{session_id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveSessionRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).GetSession
		},
	})
}
