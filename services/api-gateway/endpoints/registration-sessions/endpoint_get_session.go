package regsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to get a registration session.
type GetSessionRequest struct {
	// Session ID.
	SessionID string `json:"-" path:"session_id" validate:"required"`
}

type GetSessionEndpoint struct{}

func (e *GetSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetSessionRequest, *apiresource.RegistrationSession] {
	return &apiendpoint.APIEndpoint[*GetSessionRequest, *apiresource.RegistrationSession]{
		Title:             "Get Registration Session",
		Description:       "Returns a registration session by ID, including its current step and associated user and account details.",
		Method:            http.MethodGet,
		Route:             "/v1/auth/registration-sessions/{session_id}",
		ContentType:       "application/json",
		Request:           &GetSessionRequest{},
		Response:          &apiresource.RegistrationSession{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetSessionRequest) (*apiresource.RegistrationSession, *apierror.APIError) {
			return svc.(RegistrationSessionSvc).GetSession
		},
	}
}
