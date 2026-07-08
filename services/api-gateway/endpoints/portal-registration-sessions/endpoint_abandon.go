package portalregsessionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to abandon a portal registration session.
type AbandonPortalRegistrationSessionRequest struct {
	// Portal registration session ID.
	ID string `path:"id" validate:"required"`
}

// Abandons the buyer's in-progress registration session, so it is no longer resumed. A completed session cannot be abandoned.
type AbandonPortalRegistrationSessionEndpoint struct{}

func (e *AbandonPortalRegistrationSessionEndpoint) Materialize() *apiendpoint.APIEndpoint[*AbandonPortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession] {
	return (&apiendpoint.APIEndpoint[*AbandonPortalRegistrationSessionRequest, *apiresource.PortalRegistrationSession]{
		Title:             "Abandon Portal Registration Session",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/portal-registration-sessions/{id}/actions/abandon",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePortalRegistrationSession,
		ServiceHandler: func(svc any) func(ctx context.Context, req *AbandonPortalRegistrationSessionRequest) (*apiresource.PortalRegistrationSession, *apierror.APIError) {
			return svc.(PortalRegistrationSessionSvc).Abandon
		},
	})
}
