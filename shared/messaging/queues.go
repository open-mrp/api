package messaging

import (
	"encoding/json"

	"github.com/augno/api/shared/constants"
)

// Queue name constants define the AMQP queue names used across all services. Each
// queue is bound to the application exchange ("app") with a routing key matching its
// name. The naming convention is "{service}_{cmd|event}_{action}" to make ownership
// and intent clear at a glance.
//
// Command queues ("cmd") carry instructions that should trigger side effects (e.g.
// sending an email). Event queues ("event") carry facts about things that already
// happened (e.g. an email was sent).
const (
	// NotificationCmdSendEmailQueue carries send-email commands to the
	// notification-service. Messages on this queue contain an EmailSendData payload
	// and trigger an outbound email via SES.
	NotificationCmdSendEmailQueue = "notification_cmd_send_email"

	// NotifyEmailStatusQueue carries email delivery status updates (bounces,
	// complaints, delivery confirmations) back from SES via SNS webhook. The
	// notification-service consumes these to update email delivery records.
	NotifyEmailStatusQueue = "notify_email_status"

	// NotificationEventEmailLogQueue carries email-logged events emitted by the
	// notification-service after successfully sending an email. Downstream consumers
	// use these to maintain email audit records.
	NotificationEventEmailLogQueue = "notification_event_email_log"

	// LoggingEventRequestLogQueue carries request-log events for centralized logging.
	LoggingEventRequestLogQueue = "logging_event_request_log"

	// PlatformEventAuditLogQueue carries audit events produced by application
	// services and needs to be persisted by platform-service.
	PlatformEventAuditLogQueue = "platform_event_audit_log"

	// CoreCmdPurgeAccountDataQueue carries purge-account-data commands to the
	// core-service. Messages on this queue trigger deletion of all account-scoped
	// data across ~50 tables for a deleted sandbox account.
	CoreCmdPurgeAccountDataQueue = "core_cmd_purge_account_data"

	// CoreCmdSeedSandboxQueue carries seed-sandbox commands to the core-service.
	// Messages on this queue trigger population of a sandbox account with
	// tutorial seed data.
	CoreCmdSeedSandboxQueue = "core_cmd_seed_sandbox"

	// CoreCmdExecuteProductionStepQueue carries execute-production-step commands
	// to the core-service. Messages on this queue trigger inventory updates and
	// reservation management after batch mutations (initialize, move, merge, split).
	CoreCmdExecuteProductionStepQueue = "core_cmd_execute_production_step"

	// CoreEventSalesOrderCreatedQueue carries sales-order-created events back to
	// the core-service for out-of-band processing (e.g. CRM sync). Messages on
	// this queue contain a SalesOrderCreatedData payload.
	CoreEventSalesOrderCreatedQueue = "core_event_sales_order_created"

	// BillingEventStripeWebhookQueue carries verified Stripe webhook events for
	// asynchronous processing by the billing-service. The raw event payload and
	// metadata are enqueued immediately on receipt so the webhook endpoint can
	// return as fast as possible.
	BillingEventStripeWebhookQueue = "billing_event_stripe_webhook"

	// AgentCmdExecuteRunQueue carries execute-run commands to the agent-service.
	// Messages trigger an agent run for a specific account and agent configuration.
	AgentCmdExecuteRunQueue = "agent_cmd_execute_run"

	// AgentCmdProcessEmailQueue carries inbound-email commands to the agent-service.
	// Messages reference an S3 object containing the raw email for agent processing.
	AgentCmdProcessEmailQueue = "agent_cmd_process_email"

	// AgentCmdExecuteActionQueue carries execute-action commands to the agent-service.
	// Messages trigger execution of a proposed agent action after optional human review.
	AgentCmdExecuteActionQueue = "agent_cmd_execute_action"

	// AgentCmdContinueRunQueue carries continue-run commands to the agent-service.
	// Messages trigger continuation of an agent run that is awaiting user input.
	AgentCmdContinueRunQueue = "agent_cmd_continue_run"

	// AgentEventRunCompletedQueue carries run-completed events emitted by the
	// agent-service after an agent run finishes. Downstream consumers use these
	// to aggregate token usage and billing.
	AgentEventRunCompletedQueue = "agent_event_run_completed"

	// AgentEventRunStepQueue is the base name for the queue that carries individual
	// run step events for real-time WebSocket streaming. Each API gateway instance
	// appends a unique suffix to create its own exclusive auto-delete queue so that
	// every instance receives every event via RabbitMQ fanout.
	AgentEventRunStepQueue = "agent_event_run_step"

	// BillingCmdSyncSeatsQueue carries sync-seats commands to the billing-service.
	// Messages on this queue trigger a seat count reconciliation with Stripe.
	BillingCmdSyncSeatsQueue = "billing_cmd_sync_seats"

	// BillingCmdReportSeatChangeQueue carries report-seat-change commands to the
	// billing-service. Messages on this queue trigger a usage meter report to Stripe.
	BillingCmdReportSeatChangeQueue = "billing_cmd_report_seat_change"

	// DeadLetterQueue is the catch-all queue for messages that could not be processed
	// after exhausting retries. It is bound to the dead-letter exchange ("dlx") so
	// rejected or expired messages from any queue land here for manual inspection.
	DeadLetterQueue = "dead_letter_queue"
)

// EmailSendData is the payload for NotificationCmdSendEmailQueue messages. It
// describes a single outbound email: the recipients, subject, template, and any
// template parameters. It is serialized into the contracts.AmqpMessage.Data field
// before being written to the outbox table.
type EmailSendData struct {
	// To is the list of recipient email addresses.
	To []string `json:"to"`
	// Subject is the email subject line.
	Subject string `json:"subject"`
	// TemplateID identifies which SES email template to render.
	TemplateID constants.EmailTemplate `json:"template_id"`
	// Params are key-value pairs passed to the template engine for variable
	// substitution (e.g. user name, verification link).
	Params map[string]any `json:"params,omitempty"`
	// SendAs overrides the default sender address (e.g. "support@augno.com").
	// When nil the notification-service uses its configured default sender.
	SendAs *string `json:"send_as,omitempty"`
	// AccountID is the account context for the email, used for audit logging.
	AccountID *string `json:"account_id,omitempty"`
	// SentByID is the agent who triggered the email, used for audit logging.
	SentByID *string `json:"sent_by_id,omitempty"`
	// AttachmentData is the base64-encoded attachment content. When present,
	// the notification-service sends a raw MIME email with the attachment.
	AttachmentData *string `json:"attachment_data,omitempty"`
	// AttachmentFilename is the filename for the attachment.
	AttachmentFilename *string `json:"attachment_filename,omitempty"`
	// AttachmentContentType is the MIME content type for the attachment.
	AttachmentContentType *string `json:"attachment_content_type,omitempty"`
}

// EmailLogData is the payload for NotificationEventEmailLogQueue messages. It
// carries the metadata needed to create an email audit record after the
// notification-service has successfully dispatched an email through SES.
type EmailLogData struct {
	// SesMessageID is the unique message identifier returned by SES, used to
	// correlate delivery status events (bounces, complaints) back to this email.
	SesMessageID string `json:"ses_message_id"`
	// AccountID is the account context for audit logging.
	AccountID *string `json:"account_id,omitempty"`
	// SentByID is the agent who triggered the email for audit logging.
	SentByID *string `json:"sent_by_id,omitempty"`
	// Subject is the email subject line, stored in the audit record for quick
	// reference without needing to look up the original template.
	Subject string `json:"subject"`
	// Filename is the name of any attachment included with the email.
	Filename *string `json:"filename,omitempty"`
}

// AgentExecuteRunData is the payload for AgentCmdExecuteRunQueue messages.
// It identifies which agent config to run for which account.
type AgentExecuteRunData struct {
	AgentRunID    string `json:"agent_run_id"`
	AgentConfigID string `json:"agent_config_id"`
	AccountID     string `json:"account_id"`
	TriggerType   string `json:"trigger_type"`
}

// AgentProcessEmailData is the payload for AgentCmdProcessEmailQueue messages.
// It references an inbound email stored in S3 for agent processing.
type AgentProcessEmailData struct {
	S3Bucket  string `json:"s3_bucket"`
	S3Key     string `json:"s3_key"`
	Recipient string `json:"recipient"`
	Sender    string `json:"sender"`
	Subject   string `json:"subject"`
}

// AgentExecuteActionData is the payload for AgentCmdExecuteActionQueue messages.
// It carries a proposed action for execution after optional human review.
type AgentExecuteActionData struct {
	AgentActionID   string          `json:"agent_action_id"`
	ToolSlug        string          `json:"tool_slug"`
	ProposedPayload json.RawMessage `json:"proposed_payload"`
	AccountID       string          `json:"account_id"`
}

// AgentContinueRunData is the payload for AgentCmdContinueRunQueue messages.
// It carries the run ID, account ID, and user message for continuing a run.
type AgentContinueRunData struct {
	AgentRunID        string   `json:"agent_run_id"`
	AccountID         string   `json:"account_id"`
	Message           string   `json:"message"`
	ApprovedToolSlugs []string `json:"approved_tool_slugs,omitempty"`
	AllowedToolSlugs  []string `json:"allowed_tool_slugs,omitempty"`
	ActorID           string   `json:"actor_id,omitempty"`
	ActorType         string   `json:"actor_type,omitempty"`
	ActorName         string   `json:"actor_name,omitempty"`
}

// AgentRunStepData is the payload for AgentEventRunStepQueue messages.
// It carries a single run step event for real-time WebSocket streaming.
type AgentRunStepData struct {
	AgentRunID string          `json:"agent_run_id"`
	AccountID  string          `json:"account_id"`
	EventID    string          `json:"event_id"`
	StepType   string          `json:"step_type"`
	Title      string          `json:"title"`
	Content    *string         `json:"content,omitempty"`
	Sequence   int             `json:"sequence"`
	DurationMs *int32          `json:"duration_ms,omitempty"`
	ActionID   *string         `json:"action_id,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  string          `json:"created_at"`
	ActorID    string          `json:"actor_id,omitempty"`
	ActorType  string          `json:"actor_type,omitempty"`
	ActorName  string          `json:"actor_name,omitempty"`
}

// SeatSyncData is the payload for BillingCmdSyncSeatsQueue messages.
// It identifies the account whose seat count should be reconciled with the billing provider.
type SeatSyncData struct {
	// AccountID is the account whose seat count changed.
	AccountID string `json:"account_id"`
}

// SeatChangeReportData is the payload for BillingCmdReportSeatChangeQueue messages.
// It identifies the account whose seat count change should be reported to the billing provider's usage meters.
type SeatChangeReportData struct {
	// AccountID is the account whose seat count changed.
	AccountID string `json:"account_id"`
}

// SalesOrderCreatedData is the payload for CoreEventSalesOrderCreatedQueue messages.
// It identifies a newly created sales order so consumers can run out-of-band side
// effects (e.g. CRM sync). Consumers re-fetch the full order by ID when they need
// more than these identifiers.
type SalesOrderCreatedData struct {
	// SalesOrderID is the type-prefixed ID of the created order (e.g. "so_...").
	SalesOrderID string `json:"sales_order_id"`
	// AccountID is the owner/seller account the order belongs to.
	AccountID string `json:"account_id"`
	// BuyerAccountID is the customer account the order was created for.
	BuyerAccountID string `json:"buyer_account_id"`
	// Number is the human-facing order number.
	Number string `json:"number"`
	// StatusCode is the order's status at creation (e.g. "estimate").
	StatusCode string `json:"status_code"`
}

// AgentRunCompletedData is the payload for AgentEventRunCompletedQueue messages.
// It carries token usage and model metadata for billing aggregation.
type AgentRunCompletedData struct {
	AgentRunID       string `json:"agent_run_id"`
	AccountID        string `json:"account_id"`
	BillingAccountID string `json:"billing_account_id"`
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	LLMProvider      string `json:"llm_provider"`
	LLMModel         string `json:"llm_model"`
}
