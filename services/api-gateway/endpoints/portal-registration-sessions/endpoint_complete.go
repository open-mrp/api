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

// Completes the buyer's registration and registers them as a customer of the seller from the session's saved data.
//
// What happens depends on the session's `is_existing_customer` flag. When it is set, the buyer is linked to the seller's existing customer matching the saved `customer_number`. Otherwise a new customer is created — which requires a customer name, a billing address, a customer group, a payment term, and a shipping term in the session data — and is assigned the seller's next customer number. Either way the buyer's user is attached to that customer and the seller's customer-service team is notified.
//
// Completing an already-completed session returns it unchanged; an abandoned session cannot be completed.
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
