package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to check whether the calling customer can contact support.
//
// The customer is derived from the authenticated relation actor.
type SupportAvailabilityRequest struct{}

// Reports whether the calling customer can contact support.
//
// `available` is true only when the vendor has configured a support route with at least one recipient. The customer portal gates the contact-support feature on this so customers never open a thread no one is set up to receive.
type SupportAvailabilityEndpoint struct{}

func (e *SupportAvailabilityEndpoint) Materialize() *apiendpoint.APIEndpoint[*SupportAvailabilityRequest, *apiresource.SupportAvailability] {
	return (&apiendpoint.APIEndpoint[*SupportAvailabilityRequest, *apiresource.SupportAvailability]{
		Title:             "Support Availability",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/messaging/support-availability",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		Extras:            apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ObjectType:        constants.ObjectTypeSupportAvailability,
		// Parity with Contact Support: this probes the same create-a-support-thread capability, so the same relation actors who can open support can check availability first.
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *SupportAvailabilityRequest) (*apiresource.SupportAvailability, *apierror.APIError) {
			return svc.(ConversationSvc).SupportAvailability
		},
	})
}
