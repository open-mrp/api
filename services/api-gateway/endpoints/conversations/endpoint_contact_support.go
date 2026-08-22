package conversationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to open (or resume) the calling customer's support conversation.
//
// The customer is derived from the authenticated relation actor.
type ContactSupportRequest struct{}

// Returns the calling customer's support case with the vendor, opening it on first contact.
//
// A customer has exactly one support case, so repeat calls return the same thread rather than opening another. Opening the first case is refused when the vendor has not configured a support route with at least one recipient — check Support Availability before offering the feature. Once the case exists, the vendor's designated support staff are seated in it so the customer's first message reaches someone.
type ContactSupportEndpoint struct{}

func (e *ContactSupportEndpoint) Materialize() *apiendpoint.APIEndpoint[*ContactSupportRequest, *apiresource.Conversation] {
	return (&apiendpoint.APIEndpoint[*ContactSupportRequest, *apiresource.Conversation]{
		Title:               "Contact Support",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/support",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		ObjectType:          constants.ObjectTypeConversation,
		IncludeConfig:       conversationIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ContactSupportRequest) (*apiresource.Conversation, *apierror.APIError) {
			return svc.(ConversationSvc).ContactSupport
		},
	})
}
