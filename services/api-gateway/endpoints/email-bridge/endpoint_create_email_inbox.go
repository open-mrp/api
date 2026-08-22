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

// Request to provision a routable inbox on a verified domain.
type CreateEmailInboxRequest struct {
	// The verified domain this inbox belongs to.
	EmailDomainID string `json:"email_domain_id" validate:"required"`
	// The full inbox address (e.g. `support@acme.com`).
	//
	// Its domain part must match the selected domain, which must already be verified. The address is lowercased before it is stored, and it must not already be in use by another inbox.
	Address string `json:"address" validate:"required"`
	// Display name for the `From` header of outbound mail.
	FromName field.Optional[string] `json:"from_name,omitzero"`
	// The agent to bind to this inbox to handle incoming mail.
	//
	// With no agent bound, mail is still threaded into a conversation for your team, but nothing runs on it automatically.
	AgentConfigID field.Optional[string] `json:"agent_config_id,omitzero"`
	// How the bound agent decides whether to run on incoming mail.
	//
	// - `mention`: runs only when the agent is @mentioned, matched against the trigger keywords below.
	// - `keyword`: runs when the message contains any of the trigger keywords.
	// - `always`: runs on every incoming message.
	//
	// Leaving this unset makes the agent run on every incoming message, since email has no reliable @mention convention.
	AgentTriggerPolicy field.Optional[string] `json:"agent_trigger_policy,omitzero"`
	// The keywords that decide whether the agent runs on an incoming message.
	//
	// Under the `keyword` policy a keyword matches anywhere in the message; under `mention` it only counts where it is prefixed with `@`.
	AgentTriggerKeywords []string `json:"agent_trigger_keywords,omitzero"`
	// The messaging group (roster) whose members are seated on every conversation this inbox opens.
	//
	// Must name a group in your own account. Agents in the group are seated to run only when @mentioned, so they do not all fire alongside the inbox's own agent.
	GroupID field.Optional[string] `json:"group_id,omitzero"`
}

var sampleCreateEmailInboxFromName = "Acme Support"

var sampleCreateEmailInboxRequest = &CreateEmailInboxRequest{
	EmailDomainID:        apiresource.SampleEmailDomainID,
	Address:              "support@acme.com",
	FromName:             field.Some(sampleCreateEmailInboxFromName),
	AgentConfigID:        field.Some(apiresource.SampleAgentDefinitionID),
	AgentTriggerPolicy:   field.Some(string(constants.AgentTriggerPolicyKeyword)),
	AgentTriggerKeywords: []string{"invoice", "refund"},
}

func (*CreateEmailInboxRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateEmailInboxRequest)
}

// Provisions a routable inbox address on a verified domain.
//
// Once created, mail arriving at the address opens a customer case conversation and seats the bound agent and the group's members on it; a reply in a thread that already opened one joins that conversation instead.
type CreateEmailInboxEndpoint struct{}

func (e *CreateEmailInboxEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateEmailInboxRequest, *apiresource.EmailInbox] {
	return (&apiendpoint.APIEndpoint[*CreateEmailInboxRequest, *apiresource.EmailInbox]{
		Title:               "Create Email Inbox",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/email-inboxes",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeEmailInbox,
		IncludeConfig:       emailInboxIncludeConfig(),
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateEmailInboxRequest) (*apiresource.EmailInbox, *apierror.APIError) {
			return svc.(EmailBridgeSvc).CreateInbox
		},
	})
}
