package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to edit an email inbox's from-name, status, and default agent trigger config.
type UpdateEmailInboxRequest struct {
	// Email inbox ID.
	ID string `path:"id" validate:"required"`
	// Whether the inbox routes mail.
	//
	// - `active`: inbound mail is threaded and outbound replies are allowed.
	// - `disabled`: the inbox stays provisioned but does not route mail.
	Status string `json:"status" validate:"required"`
	// Display name for the `From` header of outbound mail.
	FromName field.Optional[string] `json:"from_name,omitzero"`
	// The agent to bind to this inbox to handle incoming mail.
	AgentConfigID field.Optional[string] `json:"agent_config_id,omitzero"`
	// How the bound agent decides whether to run on incoming mail.
	//
	// - `mention`: runs only when the agent is @mentioned in the message.
	// - `keyword`: runs when the message contains any of the trigger keywords.
	// - `always`: runs on every incoming message.
	AgentTriggerPolicy field.Optional[string] `json:"agent_trigger_policy,omitzero"`
	// Keywords that fire the agent when the trigger policy is `keyword`.
	AgentTriggerKeywords []string `json:"agent_trigger_keywords,omitzero"`
}

var sampleUpdateEmailInboxRequest = &UpdateEmailInboxRequest{
	Status: "active",
}

func (*UpdateEmailInboxRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateEmailInboxRequest)
}

// Edits an email inbox's from-name, status, and default agent trigger config.
type UpdateEmailInboxEndpoint struct{}

func (e *UpdateEmailInboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateEmailInboxRequest, *apiresource.EmailInbox] {
	return (&apiendpoint.APIEndpoint[*UpdateEmailInboxRequest, *apiresource.EmailInbox]{
		Title:               "Update Email Inbox",
		Method:              http.MethodPatch,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-inboxes/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailInbox,
		IncludeConfig:       emailInboxIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionUpdate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError) {
			return svc.(EmailBridgeSvc).UpdateInbox
		},
	})
}
