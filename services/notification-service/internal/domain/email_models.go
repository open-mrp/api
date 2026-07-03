package domain

import "time"

// Email inbox bridge domain models. An EmailDomain is a customer-owned domain verified with SES via DKIM; an EmailInbox is a routable address on that domain; an EmailMessage is the per-rfc822-message ledger row that threads inbound/outbound mail onto a conversation and dedupes redelivery.

const (
	EmailDomainStatusPending  = "pending"
	EmailDomainStatusVerified = "verified"
	EmailDomainStatusFailed   = "failed"

	EmailInboxStatusActive   = "active"
	EmailInboxStatusDisabled = "disabled"

	EmailDirectionInbound  = "inbound"
	EmailDirectionOutbound = "outbound"
)

type EmailDomain struct {
	ID         string
	AccountID  string
	Domain     string
	Status     string
	DkimTokens []string
	VerifiedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type EmailInbox struct {
	ID                   string
	AccountID            string
	EmailDomainID        string
	Address              string
	FromName             *string
	Status               string
	AgentConfigID        *string
	AgentTriggerPolicy   *string
	AgentTriggerKeywords []string
	// GroupID is the reusable roster (messaging_group) whose members are seated as participants on each new email thread this inbox opens. Nil = no team seeded (only the bound agent, if any).
	GroupID   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EmailMessage struct {
	ID             string
	AccountID      string
	ConversationID string
	MessageID      string
	EmailInboxID   string
	Direction      string
	RfcMessageID   string
	InReplyTo      *string
	References     *string
	FromAddr       string
	ToAddrs        string
	CcAddrs        *string
	Subject        *string
	RawS3Key       *string
	SesMessageID   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateEmailDomainInput captures a new domain registration. DkimTokens are the SES-issued CNAME tokens the customer must publish; they are stored so the UI can re-display them until verified.
type CreateEmailDomainInput struct {
	Domain     string
	DkimTokens []string
}

type CreateEmailInboxInput struct {
	EmailDomainID        string
	Address              string
	FromName             *string
	AgentConfigID        *string
	AgentTriggerPolicy   *string
	AgentTriggerKeywords []string
	GroupID              *string
}

type UpdateEmailInboxInput struct {
	FromName             *string
	Status               string
	AgentConfigID        *string
	AgentTriggerPolicy   *string
	AgentTriggerKeywords []string
	GroupID              *string
}

// CreateEmailMessageInput records one rfc822 message against a conversation. The unique RfcMessageID collapses at-least-once redelivery to a no-op.
type CreateEmailMessageInput struct {
	ID             string
	AccountID      string
	ConversationID string
	MessageID      string
	EmailInboxID   string
	Direction      string
	RfcMessageID   string
	InReplyTo      *string
	References     *string
	FromAddr       string
	ToAddrs        string
	CcAddrs        *string
	Subject        *string
	RawS3Key       *string
	SesMessageID   *string
}

// EmailThreadMatch is the conversation an inbound mail threads onto, resolved from its In-Reply-To / References headers.
type EmailThreadMatch struct {
	ConversationID string
	EmailInboxID   string
}

// SendInboxReplyInput drives an agent's outbound email reply on an email-bridged conversation. The recipient + threading are derived from the conversation's latest inbound email (the agent can't redirect the reply), so only the content + optional cc are supplied.
type SendInboxReplyInput struct {
	ConversationID string
	Subject        string
	Body           string
	Cc             []string
	AgentConfigID  string // the agent participant authoring the reply
	AgentRunID     string
}

// PostReplyDraftInput proposes an agent's customer reply as a real status=draft message on a customer
// case, held for human approval. Channel-agnostic: the service resolves email vs portal from the case's
// inbox binding. Distinct from PostEmailDraftInput, which posts a cosmetic "📝 Draft email" note.
type PostReplyDraftInput struct {
	ConversationID string
	Body           string
	// Subject applies only when the case is email-bridged (the outbound subject); ignored otherwise.
	Subject               string
	AgentConfigID         string // the agent participant authoring the draft
	AgentRunID            string
	SourceThreadMessageID string // the internal note the draft was composed from (provenance)
}

// IngestInboundEmailInput is the parsed inbound email the consumer hands to the conversation service to thread, dedup, persist as a message, and dispatch to the inbox's agent.
type IngestInboundEmailInput struct {
	// Recipients are all candidate delivery addresses parsed from the mail (To, Cc, Delivered-To,
	// X-Original-To), lowercased and de-duplicated. Ingestion resolves the inbox by matching these — by
	// inbox address, or by the per-inbox forwarding address (<inbox_id>@<inbound domain>) when forwarded —
	// rather than trusting a single "delivered" header, which a forwarding hop can rewrite to its own target.
	Recipients   []string
	From         string // sender email address
	FromName     string // sender display name (may be empty)
	Subject      string
	TextBody     string   // best-effort plain-text body
	RfcMessageID string   // rfc822 Message-ID; dedup key
	InReplyTo    string   // rfc822 In-Reply-To (may be empty)
	References   []string // rfc822 References message-ids (may be empty)
	RawS3Key     string   // S3 key of the raw .eml
}
