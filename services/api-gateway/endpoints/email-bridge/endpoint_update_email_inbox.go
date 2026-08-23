package emailbridgeep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to edit an email inbox's from-name, status, agent configuration, and roster.
type UpdateEmailInboxRequest struct {
	// Email inbox ID.
	ID string `path:"id" validate:"required"`
	// Whether the inbox accepts mail.
	//
	// - `active`: inbound mail is threaded into a conversation.
	// - `disabled`: the inbox stays provisioned and keeps its history, but inbound mail is dropped without being threaded.
	Status constants.EmailInboxStatus `json:"status" validate:"required"`
	// Display name for the `From` header of outbound mail.
	FromName field.Optional[string] `json:"from_name,omitzero"`
	// The agent to bind to this inbox to handle incoming mail.
	AgentConfigID field.Optional[string] `json:"agent_config_id,omitzero"`
	// How the bound agent decides whether to run on incoming mail.
	//
	// - `mention`: runs only when the agent is @mentioned, matched against the trigger keywords below.
	// - `keyword`: runs when the message contains any of the trigger keywords.
	// - `always`: runs on every incoming message.
	//
	// While no policy has been set, the agent runs on every incoming message, since email has no reliable @mention convention.
	AgentTriggerPolicy field.Optional[constants.AgentTriggerPolicy] `json:"agent_trigger_policy,omitzero"`
	// The keywords that decide whether the agent runs on an incoming message.
	//
	// Under the `keyword` policy a keyword matches anywhere in the message; under `mention` it only counts where it is prefixed with `@`.
	AgentTriggerKeywords []string `json:"agent_trigger_keywords,omitzero"`
	// The messaging group (roster) whose members are seated on every conversation this inbox opens.
	//
	// Must name a group in your own account. Changing it only affects conversations opened afterwards.
	GroupID field.Optional[string] `json:"group_id,omitzero"`
}

var sampleUpdateEmailInboxFromName = "Acme Support"

var sampleUpdateEmailInboxRequest = &UpdateEmailInboxRequest{
	Status:               "active",
	FromName:             field.Some(sampleUpdateEmailInboxFromName),
	AgentConfigID:        field.Some(apiresource.SampleAgentDefinitionID),
	AgentTriggerPolicy:   field.Some(constants.AgentTriggerPolicyKeyword),
	AgentTriggerKeywords: []string{"invoice", "refund"},
}

func (*UpdateEmailInboxRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateEmailInboxRequest)
}

// Edits an email inbox's from-name, status, agent configuration, and roster.
//
// Every field except `status` is merged into the inbox's current settings: a field you omit — and an empty array you send — keeps the value it already has, so this endpoint can change a setting but cannot clear one back to unset. The inbox's address and domain are fixed at creation and cannot be changed here.
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
