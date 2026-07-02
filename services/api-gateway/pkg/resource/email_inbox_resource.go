package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleEmailInboxID = "eminb_018e88072d1320808dc9bbb02"

// A routable email inbox on a verified domain.
//
// Inbound mail to this address is threaded into a chat conversation, and outbound replies may be sent from this identity. The optional agent trigger config controls whether the bound agent runs automatically on incoming mail.
type EmailInbox struct {
	// Email inbox ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=email_inbox"`
	// Whether the inbox is currently routing mail.
	//
	// - `active`: inbound mail is threaded and outbound replies are allowed.
	// - `disabled`: the inbox is provisioned but drops inbound mail and does not send replies.
	Status string `json:"status" validate:"required"`
	// The domain this inbox belongs to.
	EmailDomain *EmailDomain `json:"email_domain" validate:"required" expandable:"true"`
	// The full inbox address (e.g. `support@acme.com`).
	Address string `json:"address" validate:"required"`
	// The display name used in the `From` header of outbound mail.
	FromName *string `json:"from_name"`
	// The agent that handles mail for this inbox, when one is bound.
	//
	// `null` when no agent is bound.
	AgentConfig *AgentDefinition `json:"agent_config" expandable:"true"`
	// When the bound agent runs on incoming mail.
	//
	// - `mention`: only when the agent is @mentioned, matched against its trigger keywords.
	// - `keyword`: when the mail contains any of the configured trigger keywords.
	// - `always`: on every incoming message.
	AgentTriggerPolicy *string `json:"agent_trigger_policy"`
	// Keywords that fire the agent when `agent_trigger_policy` is `keyword`.
	AgentTriggerKeywords []string `json:"agent_trigger_keywords"`
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
	EmailDomain:        &EmailDomain{ID: SampleEmailDomainID, Object: constants.ObjectTypeEmailDomain},
	Address:            "support@acme.com",
	FromName:           &sampleEmailInboxFromName,
	Status:             "active",
	AgentConfig:        &AgentDefinition{ID: SampleAgentDefinitionID, Object: constants.ObjectTypeAgentDefinition},
	AgentTriggerPolicy: &sampleEmailInboxTriggerPolicy,
	CreatedAt:          timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:          timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*EmailInbox) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleEmailInbox)
}
