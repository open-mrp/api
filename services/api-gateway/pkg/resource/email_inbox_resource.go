package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleEmailInboxID = "eminb_2s9kobr9s7tp"

// A routable email inbox on a verified domain.
//
// Mail sent to this address is threaded into a conversation: the first message of a thread opens a new customer case, and later messages in the same thread join the conversation it already created. Replies to the customer go back out from this address, and the bound agent — if there is one — can draft or send them.
type EmailInbox struct {
	// Email inbox ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_inbox"`
	// Whether the inbox is currently accepting mail.
	//
	// - `active`: inbound mail is threaded into a conversation.
	// - `disabled`: the inbox stays provisioned and keeps its history, but inbound mail is dropped without being threaded.
	Status string `json:"status" validate:"required"`
	// The domain this inbox belongs to.
	EmailDomain *EmailDomain `json:"email_domain" validate:"required" expandable:"true"`
	// The full inbox address (e.g. `support@acme.com`).
	Address string `json:"address" validate:"required"`
	// A forwarding address on an OpenMRP-owned domain that also routes to this inbox.
	//
	// Use this when your domain's mail is hosted elsewhere (e.g. Google Workspace, Microsoft 365) and you cannot point its MX records at OpenMRP: forward mail from `address` to this address instead, and it will still be threaded into a conversation.
	ForwardingAddress *string `json:"forwarding_address"`
	// The display name used in the `From` header of outbound mail.
	FromName *string `json:"from_name"`
	// The agent that handles mail for this inbox.
	//
	// The agent is seated on every conversation this inbox opens, so it can read the thread and draft or send replies.
	AgentConfig *AgentDefinition `json:"agent_config" expandable:"true"`
	// When the bound agent runs on incoming mail.
	//
	// - `mention`: only when the agent is @mentioned, matched against its trigger keywords.
	// - `keyword`: when the mail contains any of the configured trigger keywords.
	// - `always`: on every incoming message.
	//
	// When no policy is set the agent runs on every incoming message, since email has no reliable @mention convention.
	AgentTriggerPolicy *string `json:"agent_trigger_policy"`
	// The keywords that decide whether the agent runs on an incoming message.
	//
	// Under the `keyword` policy a keyword matches anywhere in the message; under `mention` it only counts where it is prefixed with `@`.
	AgentTriggerKeywords []string `json:"agent_trigger_keywords"`
	// The messaging group (roster) whose members are added to every conversation this inbox opens.
	//
	// Its members join each new email thread so the team can read, edit, and approve replies alongside the bound agent. Membership is captured when the thread opens, so later edits to the group only affect conversations opened after the change.
	GroupID *string `json:"group_id"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var (
	sampleEmailInboxFromName      = "Acme Support"
	sampleEmailInboxTriggerPolicy = "always"
)

var SampleEmailInbox = &EmailInbox{
	ID:                 SampleEmailInboxID,
	Object:             constants.ObjectTypeEmailInbox,
	EmailDomain:        SampleEmailDomain,
	Address:            "support@acme.com",
	FromName:           &sampleEmailInboxFromName,
	Status:             "active",
	AgentConfig:        SampleAgentDefinition,
	AgentTriggerPolicy: &sampleEmailInboxTriggerPolicy,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*EmailInbox) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailInbox)
}
