package portalregsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to retrieve a portal registration session.
type GetPortalRegistrationSessionRequest struct {
	// Portal registration session ID.
	ID string `path:"id" validate:"required"`
}

// Returns the authenticated buyer's portal registration session, so the wizard can restore its saved step and form data. Expired or unknown sessions return a 404.
type GetPortalRegistrationSessionEndpoint struct{}

func (e *GetPortalRegistrationSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetPortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession] {
	return (&apiendpoint.APIEndpoint[*GetPortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession]{
		Title:             "Retrieve Portal Registration Session",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/sales/portal-registration-sessions/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePortalRegistrationSession,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetPortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
			return svc.(PortalRegistrationSessionSvc).Get
		},
	})
}
