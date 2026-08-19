package messaging

import (
	"encoding/json"
	"time"

	"github.com/augno/api/shared/constants"
)

// Queue name constants define the AMQP queue names used across all services. Each queue is bound to the application exchange ("app") with a routing key matching its name. The naming convention is "{service}_{cmd|event}_{action}" to make ownership and intent clear at a glance.
//
// Command queues ("cmd") carry instructions that should trigger side effects (e.g. sending an email). Event queues ("event") carry facts about things that already happened (e.g. an email was sent).
const (
	// NotificationCmdSendEmailQueue carries send-email commands to the notification-service. Messages on this queue contain an EmailSendData payload and trigger an outbound email via SES.
	NotificationCmdSendEmailQueue = "notification_cmd_send_email"

	// NotifyEmailStatusQueue carries email delivery status updates (bounces, complaints, delivery confirmations) back from SES via SNS webhook. The notification-service consumes these to update email delivery records.
	NotifyEmailStatusQueue = "notify_email_status"

	// NotificationEventEmailLogQueue carries email-logged events emitted by the notification-service after successfully sending an email. Downstream consumers use these to maintain email audit records.
	NotificationEventEmailLogQueue = "notification_event_email_log"

	// LoggingEventRequestLogQueue carries request-log events for centralized logging.
	LoggingEventRequestLogQueue = "logging_event_request_log"

	// PlatformEventAuditLogQueue carries audit events produced by application services and needs to be persisted by platform-service.
	PlatformEventAuditLogQueue = "platform_event_audit_log"

	// CoreCmdPurgeAccountDataQueue carries purge-account-data commands to the core-service. Messages on this queue trigger deletion of all account-scoped data across ~50 tables for a deleted sandbox account.
	CoreCmdPurgeAccountDataQueue = "core_cmd_purge_account_data"

	// CoreCmdSeedSandboxQueue carries seed-sandbox commands to the core-service. Messages on this queue trigger population of a sandbox account with tutorial seed data.
	CoreCmdSeedSandboxQueue = "core_cmd_seed_sandbox"

	// CoreCmdExecuteProductionStepQueue carries execute-production-step commands to the core-service. Messages on this queue trigger inventory updates and reservation management after batch mutations (initialize, move, merge, split).
	//
	// Superseded by CoreEventBatchScannedInventoryQueue. Kept so commands already enqueued when the
	// switch happens still drain; delete once it has been empty across a deploy.
	CoreCmdExecuteProductionStepQueue = "core_cmd_execute_production_step"

	// CoreEventInventoryReceivedAllocationQueue carries inventory-received events to the core-service consumer that offers the new stock to whatever demand went short waiting for it.
	CoreEventInventoryReceivedAllocationQueue = "core_event_inventory_received_allocation"

	// CoreEventItemCostBasisChangedQueue carries cost-basis-changed events to the core-service consumer that recomputes the cost of every item downstream of the change.
	CoreEventItemCostBasisChangedQueue = "core_event_item_cost_basis_changed_costing"

	// CoreEventBatchScannedInventoryQueue carries batch-scanned events to the core-service consumer that moves inventory: the produced receipt, the reservations seconds and waste release, and the material consumption.
	//
	// The queue is named for what its consumer does, not for the event, because a topic exchange gives every bound queue its own copy while consumers sharing a queue compete for messages. A second reaction to the same scan — crediting schedule attainment, say — binds CoreEventBatchScanned on a queue of its own and both run.
	CoreEventBatchScannedInventoryQueue = "core_event_batch_scanned_inventory"

	// carries undo-batch-scan commands to the core-service, reversing the receipts, issues, and reservations a scan recorded against a deleted batch.
	CoreCmdUndoBatchScanQueue = "core_cmd_undo_batch_scan"

	// CoreCmdSyncStripeCustomerQueue carries sync-stripe-customer commands to the core-service, which creates or updates the customer's counterpart in the account's connected Stripe integration. Messages contain a SyncStripeCustomerEvent payload.
	CoreCmdSyncStripeCustomerQueue = "core_cmd_sync_stripe_customer"

	// Async bulk operation command queues ("core_cmd_<slug>") are derived from their
	// canonical identity — see BulkOperation.Queue in bulk_operations.go, the single source
	// of truth for each operation's queue, routing key, and inbox handler key.

	// carries a request to solve and persist one production schedule version. The cadence tick only enqueues onto this queue: a solve takes minutes on a real tenant, and doing it inside the scheduler lease would block every other account behind whichever one is currently solving.
	CoreCmdGenerateProductionScheduleQueue = "core_cmd_generate_production_schedule"

	// carries sales-order-created events back to the core-service for out-of-band processing (e.g. CRM sync). Messages on this queue contain a SalesOrderCreatedData payload.
	CoreEventSalesOrderCreatedQueue = "core_event_sales_order_created"

	// CoreEventSalesOrderShippingUpdatedQueue carries sales-order shipping-changed events back to the core-service, which re-syncs the order's shipment records' carrier / service level / ship-to. Messages contain a SalesOrderShippingUpdatedData payload.
	CoreEventSalesOrderShippingUpdatedQueue = "core_event_sales_order_shipping_updated"

	// CoreEventCustomerRegisteredQueue carries customer-registered events to the notification-service, which notifies the seller's customer-service support-route group. Messages contain a CustomerRegisteredData payload.
	CoreEventCustomerRegisteredQueue = "core_event_customer_registered"

	// CoreCmdHubspotSyncQueue carries HubSpot backfill commands (preview and execute) to the core-service, which runs the long-running matching/sync passes out-of-band. Bound to both the preview and execute routing keys; the consumer dispatches on routing key. Messages contain a HubspotSyncCommandData payload.
	CoreCmdHubspotSyncQueue = "core_cmd_hubspot_sync"

	// BillingEventStripeWebhookQueue carries verified Stripe webhook events for asynchronous processing by the billing-service. The raw event payload and metadata are enqueued immediately on receipt so the webhook endpoint can return as fast as possible.
	BillingEventStripeWebhookQueue = "billing_event_stripe_webhook"

	// AgentCmdExecuteRunQueue carries execute-run commands to the agent-service. Messages trigger an agent run for a specific account and agent configuration.
	AgentCmdExecuteRunQueue = "agent_cmd_execute_run"

	// AgentCmdExecuteActionQueue carries execute-action commands to the agent-service. Messages trigger execution of a proposed agent action after optional human review.
	AgentCmdExecuteActionQueue = "agent_cmd_execute_action"

	// AgentCmdContinueRunQueue carries continue-run commands to the agent-service. Messages trigger continuation of an agent run that is awaiting user input.
	AgentCmdContinueRunQueue = "agent_cmd_continue_run"

	// AgentCmdChatRunQueue carries chat-run commands (from notification-service) to the agent-service: create a chat-linked run and execute it.
	AgentCmdChatRunQueue = "agent_cmd_chat_run"

	// AgentEventRunCompletedQueue carries run-completed events emitted by the agent-service after an agent run finishes. It is the durable, shared work queue for billing-service token/usage aggregation (exactly-once across billing replicas). The api-gateway also consumes run-completed events for WebSocket fan-out, but via its own per-instance queue (ConsumeFanout with this base name) so every gateway replica gets a copy — it must not join this shared queue, or billing and the gateway would steal each other's events.
	AgentEventRunCompletedQueue = "agent_event_run_completed"

	// AgentEventRunStepQueue is the base name for the queue that carries individual run step events for real-time WebSocket streaming. Each API gateway instance appends a unique suffix to create its own exclusive auto-delete queue so that every instance receives every event via RabbitMQ fanout.
	AgentEventRunStepQueue = "agent_event_run_step"

	// BillingCmdSyncSeatsQueue carries sync-seats commands to the billing-service. Messages on this queue trigger a seat count reconciliation with Stripe.
	BillingCmdSyncSeatsQueue = "billing_cmd_sync_seats"

	// BillingCmdReportSeatChangeQueue carries report-seat-change commands to the billing-service. Messages on this queue trigger a usage meter report to Stripe.
	BillingCmdReportSeatChangeQueue = "billing_cmd_report_seat_change"

	// NotificationCmdFanoutQueue carries alert/message fan-out intents to notification-service. It is inbox-deduped and bound to NotificationCmdFanout (and NotificationCmdSendMessage).
	NotificationCmdFanoutQueue = "notification_cmd_fanout"

	// NotificationCmdAgentReplyQueue carries an agent's chat reply (from agent-service) to notification-service. Inbox-deduped and bound to NotificationCmdAgentReply.
	NotificationCmdAgentReplyQueue = "notification_cmd_agent_reply"

	// NotificationCmdAgentReplyPatchQueue carries best-effort partial-body updates for an in-flight streaming agent reply. Not inbox-deduped (patches are idempotent last-write-wins) and bound to NotificationCmdAgentReplyPatch.
	NotificationCmdAgentReplyPatchQueue = "notification_cmd_agent_reply_patch"

	// NotificationEventDeliveredQueue is the base name for the realtime push queue consumed by api-gateway. Like AgentEventRunStepQueue, each gateway instance appends a unique suffix to create its own exclusive auto-delete queue (bound to NotificationEventDelivered and NotificationEventConversationUpdated) so every instance receives every event via RabbitMQ fanout.
	NotificationEventDeliveredQueue = "notification_event_delivered"

	// DeadLetterQueue is the catch-all queue for messages that could not be processed after exhausting retries. It is bound to the dead-letter exchange ("dlx") so rejected or expired messages from any queue land here for manual inspection.
	DeadLetterQueue = "dead_letter_queue"
)

// EmailSendData is the payload for NotificationCmdSendEmailQueue messages. It describes a single outbound email: the recipients, subject, template, and any template parameters. It is serialized into the contracts.AmqpMessage.Data field before being written to the outbox table.
type EmailSendData struct {
	// To is the list of recipient email addresses.
	To []string `json:"to"`
	// Subject is the email subject line.
	Subject string `json:"subject"`
	// TemplateID identifies which SES email template to render.
	TemplateID constants.EmailTemplate `json:"template_id"`
	// Params are key-value pairs passed to the template engine for variable substitution (e.g. user name, verification link).
	Params map[string]any `json:"params,omitempty"`
	// SendAs overrides the default sender address (e.g. "support@augno.com"). When nil the notification-service uses its configured default sender.
	SendAs *string `json:"send_as,omitempty"`
	// AccountID is the account context for the email, used for audit logging.
	AccountID *string `json:"account_id,omitempty"`
	// SentByID is the agent who triggered the email, used for audit logging.
	SentByID *string `json:"sent_by_id,omitempty"`
	// AttachmentData is the base64-encoded attachment content. When present, the notification-service sends a raw MIME email with the attachment.
	AttachmentData *string `json:"attachment_data,omitempty"`
	// AttachmentFilename is the filename for the attachment.
	AttachmentFilename *string `json:"attachment_filename,omitempty"`
	// AttachmentContentType is the MIME content type for the attachment.
	AttachmentContentType *string `json:"attachment_content_type,omitempty"`
}

// EmailLogData is the payload for NotificationEventEmailLogQueue messages. It carries the metadata needed to create an email audit record after the notification-service has successfully dispatched an email through SES.
type EmailLogData struct {
	// SesMessageID is the unique message identifier returned by SES, used to correlate delivery status events (bounces, complaints) back to this email.
	SesMessageID string `json:"ses_message_id"`
	// To are the recipient addresses, persisted as email_recipient rows so the log lists and searches by who received it.
	To []string `json:"to,omitempty"`
	// AccountID is the account context for audit logging.
	AccountID *string `json:"account_id,omitempty"`
	// SentByID is the agent who triggered the email for audit logging.
	SentByID *string `json:"sent_by_id,omitempty"`
	// Subject is the email subject line, stored in the audit record for quick reference without needing to look up the original template.
	Subject string `json:"subject"`
	// Filename is the name of any attachment included with the email.
	Filename *string `json:"filename,omitempty"`
}

// AgentExecuteRunData is the payload for AgentCmdExecuteRunQueue messages. It identifies which agent config to run for which account.
type AgentExecuteRunData struct {
	AgentRunID    string `json:"agent_run_id"`
	AgentConfigID string `json:"agent_config_id"`
	AccountID     string `json:"account_id"`
	TriggerType   string `json:"trigger_type"`
}

// AgentChatRunData is the payload for AgentCmdChatRunQueue: start an agent run from a chat message.
// agent-service creates a chat-linked run (conversation_id + trigger_message_id, trigger_type=chat, input=Message) for the agent definition and executes it; on completion the run posts its reply back into the conversation. AgentDefinitionID is the participant's agent identifier.
type AgentChatRunData struct {
	AccountID         string `json:"account_id"`
	AgentDefinitionID string `json:"agent_definition_id"`
	ConversationID    string `json:"conversation_id"`
	TriggerMessageID  string `json:"trigger_message_id"`
	Message           string `json:"message"`
	// History is the recent thread context preceding the trigger, oldest-first, so the agent can follow the conversation instead of seeing only the triggering message. Role is from this agent's perspective: its own past replies are "assistant", everyone else is "user". Only set when starting a new run — a continued run already carries its own history.
	History []ChatHistoryMessage `json:"history,omitempty"`
	// ContinueRunID, when set, is the run to continue (the user replied directly to that run's message) instead of starting a new run. Empty means start a fresh run.
	ContinueRunID string `json:"continue_run_id,omitempty"`
}

// ChatHistoryMessage is one prior turn of conversation context for a chat-triggered agent run.
type ChatHistoryMessage struct {
	// Role is "assistant" for the dispatched agent's own earlier replies, "user" for everyone else.
	Role string `json:"role"`
	// Name is the sender's display name when known (people); empty for agents.
	Name string `json:"name,omitempty"`
	// AgentConfigID is set when a *different* agent authored this turn. Its display name lives in agent-service (not resolvable in notif-service), so it's carried here and resolved into Name when the run is created. Empty for people and the dispatched agent's own turns.
	AgentConfigID string `json:"agent_config_id,omitempty"`
	Body          string `json:"body"`
}

// AgentExecuteActionData is the payload for AgentCmdExecuteActionQueue messages. It carries a proposed action for execution after optional human review.
type AgentExecuteActionData struct {
	AgentActionID   string          `json:"agent_action_id"`
	ToolSlug        string          `json:"tool_slug"`
	ProposedPayload json.RawMessage `json:"proposed_payload"`
	AccountID       string          `json:"account_id"`
}

// AgentContinueRunData is the payload for AgentCmdContinueRunQueue messages. It carries the run ID, account ID, and user message for continuing a run.
type AgentContinueRunData struct {
	AgentRunID        string   `json:"agent_run_id"`
	AccountID         string   `json:"account_id"`
	Message           string   `json:"message"`
	ApprovedToolSlugs []string `json:"approved_tool_slugs,omitempty"`
	// ApproveAllPending, when true, approves every still-pending review-gated tool on the run (the "Approve all" control). It is the only way an empty ApprovedToolSlugs grants approval — set solely on an explicit approval with no specific slugs. A typed-message continuation or a retry leaves it false, so a blocked tool is never silently let through and re-prompts the next time it is called.
	ApproveAllPending bool `json:"approve_all_pending,omitempty"`
	// RejectedToolSlugs are the review-gated tools the human denied on this resume. The run continues; the runner answers each with a synthetic "denied by user" tool result so the agent proceeds without them.
	RejectedToolSlugs []string `json:"rejected_tool_slugs,omitempty"`
	// ApprovedToolCallIDs / RejectedToolCallIDs are per-call decisions: the tool_use_ids of individual blocked
	// calls the human approved/denied. Unlike the slug lists (which apply to every pending call of a slug),
	// these target one specific call, so two same-slug calls with different inputs are decided independently.
	ApprovedToolCallIDs []string `json:"approved_tool_call_ids,omitempty"`
	RejectedToolCallIDs []string `json:"rejected_tool_call_ids,omitempty"`
	ActorID             string   `json:"actor_id,omitempty"`
	ActorType           string   `json:"actor_type,omitempty"`
	ActorName           string   `json:"actor_name,omitempty"`
	// ReplyToMessageID threads this turn's reply under the message that triggered the continuation (the user's reply to the agent), so a chat thread keeps growing as one. Empty for non-chat continuations (e.g. the agent-run console).
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
}

// AgentRunStepData is the payload for AgentEventRunStepQueue messages. It carries a single run step event for real-time WebSocket streaming.
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
	// Terminal marks this step as the run's final event (e.g. the "Run failed" error step). The WS gateway, on seeing it, also emits a terminal run_complete frame on the run topic so the frontend leaves its loading state. Successful/awaiting runs already get that frame from the run-completed event; a failed run only emits this step, so without this flag the live run view stays stuck loading until a hard refresh re-fetches the persisted "failed" status.
	Terminal bool `json:"terminal,omitempty"`
}

// SeatSyncData is the payload for BillingCmdSyncSeatsQueue messages. It identifies the account whose seat count should be reconciled with the billing provider.
type SeatSyncData struct {
	// AccountID is the account whose seat count changed.
	AccountID string `json:"account_id"`
}

// SeatChangeReportData is the payload for BillingCmdReportSeatChangeQueue messages. It identifies the account whose seat count change should be reported to the billing provider's usage meters.
type SeatChangeReportData struct {
	// AccountID is the account whose seat count changed.
	AccountID string `json:"account_id"`
}

// GenerateProductionScheduleData is the payload for CoreCmdGenerateProductionScheduleQueue messages. The schedule row already exists in `generating` status when the message is published, so the consumer solves into a row that is already visible rather than creating one — a tick that enqueued and then died would otherwise leave no trace.
type GenerateProductionScheduleData struct {
	AccountID  string `json:"account_id"`
	ScheduleID string `json:"schedule_id"`
	// PlanningAsOf is stamped by the tick, not read at consume time, so a message that sits in the queue still plans against the moment the cadence fired.
	PlanningAsOf time.Time `json:"planning_as_of"`
	// AutoPublish publishes the version as soon as it solves, for merchants who want the cadence to be the whole workflow.
	AutoPublish bool `json:"auto_publish"`
}

// SalesOrderCreatedData is the payload for CoreEventSalesOrderCreatedQueue messages. It identifies a newly created sales order so consumers can run out-of-band side effects (e.g. CRM sync). Consumers re-fetch the full order by ID when they need more than these identifiers.
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

// SalesOrderShippingUpdatedData is the payload for CoreEventSalesOrderShippingUpdatedQueue messages. It identifies an order whose carrier / service level / ship-to changed; the consumer re-fetches the order and syncs its shipment records to the order's current shipping fields.
type SalesOrderShippingUpdatedData struct {
	// SalesOrderID is the type-prefixed ID of the updated order.
	SalesOrderID string `json:"sales_order_id"`
	// AccountID is the owner/seller account the order belongs to.
	AccountID string `json:"account_id"`
}

// CustomerRegisteredData is the payload for CoreEventCustomerRegisteredQueue messages. It identifies a buyer who completed portal registration so the notification-service consumer can notify the seller's customer-service support-route group. All display fields the consumer needs are carried here — the consumer (notification-service) has no core-service client to re-fetch them.
type CustomerRegisteredData struct {
	// SellerAccountID is the seller/vendor account whose portal the buyer registered on. The support route is resolved against this account.
	SellerAccountID string `json:"seller_account_id"`
	// CustomerAccountID is the buyer's customer account (the notification links to it).
	CustomerAccountID string `json:"customer_account_id"`
	// CustomerName is the customer's display name (best-effort; may be empty for an existing-customer join, in which case the consumer falls back to the number).
	CustomerName string `json:"customer_name,omitempty"`
	// CustomerNumber is the seller-facing customer number.
	CustomerNumber string `json:"customer_number,omitempty"`
	// RegistrantUserID is the user (us_) who registered.
	RegistrantUserID string `json:"registrant_user_id,omitempty"`
	// IsExistingCustomer is true when the buyer joined an existing customer account rather than creating a new one.
	IsExistingCustomer bool `json:"is_existing_customer,omitempty"`
}

// HubspotSyncCommandData is the payload for CoreCmdHubspotSyncQueue messages (both preview and execute). It identifies the backfill job to run; the consumer dispatches on the message's routing key.
type HubspotSyncCommandData struct {
	// JobID is the type-prefixed HubSpot sync job id (e.g. "igjb_...").
	JobID string `json:"job_id"`
	// AccountID is the account the job belongs to.
	AccountID string `json:"account_id"`
}

// AgentRunCompletedData is the payload for AgentEventRunCompletedQueue messages. It carries token usage and model metadata for billing aggregation.
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

// ConversationRef points a fan-out at a conversation, either directly by ID or by the resource it is anchored to (the system channel is resolved by category when nil).
type ConversationRef struct {
	ConversationID    string `json:"conversation_id,omitempty"`
	TopicResourceType string `json:"topic_resource_type,omitempty"`
	TopicResourceID   string `json:"topic_resource_id,omitempty"`
}

// AlertFanoutData is the payload for NotificationCmdFanoutQueue messages. A producer (core/agent/billing/notification-service) emits one alert/message intent; the notification-service consumer turns it into a message plus per-recipient notification rows and a realtime push. Content may be templated (TemplateKey/TemplateParams) for i18n, with Title/Body as default-locale fallbacks.
// AgentReplyData is the payload for NotificationCmdAgentReplyQueue: an agent's reply to post into a conversation. notification-service resolves the agent participant from (ConversationID, AgentConfigID), creates a kind=agent message authored by that participant and linked to AgentRunID, and fans it out.
// ClientMessageID makes it idempotent across redelivery.
//
// A streaming reply spans two of these: Phase "start" creates the row empty and in streaming_state, then Phase "final" sets the finished body and flips it to complete (and fires the bell). MessageID is the agent-owned row id shared across start/patch/final so they target the same record. Phase "" is the legacy single-shot path (create-and-complete in one message). Failed marks a "final" that resolves a started bubble to an error message.
type AgentReplyData struct {
	AccountID      string `json:"account_id"`
	ConversationID string `json:"conversation_id"`
	AgentConfigID  string `json:"agent_config_id"`
	// AgentName is the agent definition's display name, carried so chat bell notifications can title themselves after the agent (name lives in agent-service, not notification-service).
	AgentName       string `json:"agent_name,omitempty"`
	AgentRunID      string `json:"agent_run_id"`
	Body            string `json:"body"`
	ClientMessageID string `json:"client_message_id"`
	// MessageID is the agent-generated message row id, shared by the start/patch/final messages of one streaming reply so they address the same record. Empty falls back to a service-generated id.
	MessageID string `json:"message_id,omitempty"`
	// Phase is "start" | "final" | "" (legacy single-shot create-and-complete).
	Phase string `json:"phase,omitempty"`
	// Failed marks a "final" reply that resolves an errored run (the body is the user-facing error).
	Failed bool `json:"failed,omitempty"`
	// ErrorCode is the machine-readable api-error code for a failed reply (e.g. "agent_spending_cap_reached"), carried onto the message so the client can react (e.g. prompt to raise the spending limit). Empty for non-cap or non-failed replies.
	ErrorCode string `json:"error_code,omitempty"`
	// ReplyToMessageID threads the reply under the message that triggered the run (a mention or keyword), so it renders as a reply. Empty for continuation turns (already in a reply thread).
	ReplyToMessageID string `json:"reply_to_message_id,omitempty"`
	// ApprovalEvent marks this as a tool-approval notice rather than an agent reply: the consumer posts it as a senderless system_event (Body = "{approver} approved {tools}") that renders as a timeline divider, not an agent bubble. AgentConfigID/AgentRunID are not required in this mode.
	ApprovalEvent bool `json:"approval_event,omitempty"`
}

// AgentReplyPatchData is the payload for NotificationCmdAgentReplyPatchQueue: a best-effort partial-body update for an in-flight streaming agent reply. Body is the full accumulated answer so far (not a delta), so a dropped or reordered patch never corrupts the record — the next patch or the "final" reply reconciles it. notification-service updates the row (without touching edited_at) and pushes a server-only message.updated to the conversation's live subscribers.
type AgentReplyPatchData struct {
	AccountID      string `json:"account_id"`
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
	Body           string `json:"body"`
}

type AlertFanoutData struct {
	AccountID        string           `json:"account_id"`
	Category         string           `json:"category"`
	ConversationRef  *ConversationRef `json:"conversation_ref,omitempty"`
	Kind             string           `json:"kind"` // system_event | alert | chat | agent
	Title            string           `json:"title"`
	Body             string           `json:"body,omitempty"`
	Preview          string           `json:"preview,omitempty"`
	TemplateKey      string           `json:"template_key,omitempty"`
	TemplateParams   json.RawMessage  `json:"template_params,omitempty"`
	LinkResourceType string           `json:"link_resource_type,omitempty"`
	LinkResourceID   string           `json:"link_resource_id,omitempty"`
	Priority         string           `json:"priority,omitempty"`
	// Polymorphic sender attribution (user | group | system | agent | apikey).
	SenderType string `json:"sender_type,omitempty"`
	SenderID   string `json:"sender_id,omitempty"`
	SenderName string `json:"sender_name,omitempty"`
	// RecipientAccountUserIDs is an explicit recipient list (account_user ids); empty + Broadcast=true means all active users in the account.
	RecipientAccountUserIDs []string `json:"recipient_account_user_ids,omitempty"`
	// RecipientUserIDs are user ids (us_) the fan-out resolves to account_user ids within AccountID — for producers (e.g. agent-service) that hold the user id, not the account_user id.
	RecipientUserIDs []string        `json:"recipient_user_ids,omitempty"`
	Broadcast        bool            `json:"broadcast,omitempty"`
	DedupeKey        string          `json:"dedupe_key,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

// RealtimeDeliveryData is the payload for NotificationEventDeliveredQueue messages. notification-service emits it; every api-gateway instance consumes it and fans it out to the matching Hub topic (user:<account_user_id> for the bell, conv:<conversation_id> for live chat). Best-effort: the persisted rows remain the source of truth.
type RealtimeDeliveryData struct {
	AccountID string `json:"account_id"`
	// RecipientUserID is the user id (us_) used as the WS user-topic key (the gateway subscribes user:<user_id> from the validated identity's actor id).
	RecipientUserID string `json:"recipient_user_id,omitempty"`
	// RecipientAccountUserID is the per-account recipient (acus_) the notification belongs to.
	RecipientAccountUserID string `json:"recipient_account_user_id,omitempty"`
	// ConversationID targets the per-conversation topic (live chat).
	ConversationID string `json:"conversation_id,omitempty"`
	// AnnouncementID targets the per-account broadcast topic (account:<account_id>).
	AnnouncementID string `json:"announcement_id,omitempty"`
	// Event is the client-facing event name: notification.created | announcement.created | message.created | conversation.updated | unread.changed | account.unread_hint.
	Event string `json:"event"`
	// Visibility is the audience of a message-bearing frame (internal | external | system); empty for non-message events. SAFETY: a customer-subscribed conversation socket must drop frames whose Visibility is "internal" so an internal note is never delivered to the customer. The authoritative guarantee is the visibility-filtered read path; this lets the realtime layer enforce the same.
	Visibility     string          `json:"visibility,omitempty"`
	NotificationID string          `json:"notification_id,omitempty"`
	MessageID      string          `json:"message_id,omitempty"`
	Sequence       int64           `json:"sequence,omitempty"`
	UnreadCount    *int64          `json:"unread_count,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}
