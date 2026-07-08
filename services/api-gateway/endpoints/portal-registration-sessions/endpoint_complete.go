package portalregsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to complete a portal registration session.
type CompletePortalRegistrationSessionRequest struct {
	// Portal registration session ID.
	ID string `path:"id" validate:"required"`
}

// Completes the buyer's registration: registers them as a customer of the seller from the session's saved data, then marks the session complete. Idempotent — completing an already-complete session returns it unchanged.
type CompletePortalRegistrationSessionEndpoint struct{}

func (e *CompletePortalRegistrationSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CompletePortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession] {
	return (&apiendpoint.APIEndpoint[*CompletePortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession]{
		Title:             "Complete Portal Registration Session",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/portal-registration-sessions/{id}/actions/complete",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePortalRegistrationSession,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CompletePortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
			return svc.(PortalRegistrationSessionSvc).Complete
		},
	})
}
