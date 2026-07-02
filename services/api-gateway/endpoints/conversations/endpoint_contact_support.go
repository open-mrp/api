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

// Request to open (or resume) the calling customer's support conversation.
//
// The customer is derived from the authenticated relation actor.
type ContactSupportRequest struct{}

// Returns the calling customer's portal support case (`audience=customer`), creating it on first contact.
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
