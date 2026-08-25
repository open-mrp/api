package contracts

import "github.com/open-mrp/api/services/auth-service/pkg/types"

// AmqpMessage is the envelope for all messages published to and consumed from RabbitMQ. It carries the caller's identity, the application payload, and tracing/idempotency metadata needed to correlate and deduplicate messages.
type AmqpMessage struct {
	// Identity is the authenticated caller that triggered the message.
	Identity *types.Identity `json:"identity"`
	// Data is the application-specific payload (typically JSON-encoded).
	Data []byte `json:"data"`
	// MessageID uniquely identifies this message instance.
	MessageID string `json:"message_id,omitempty"`
	// RequestID is the originating HTTP request ID for end-to-end tracing.
	RequestID string `json:"request_id,omitempty"`
	// ParentMessageID links this message to the message that caused it, forming a causal chain.
	ParentMessageID string `json:"parent_message_id,omitempty"`
	// OperationID groups related messages that belong to the same logical operation.
	OperationID string `json:"operation_id,omitempty"`
	// IdempotencyKey is forwarded from the HTTP layer so consumers can deduplicate processing.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	// Step identifies the current stage in a multi-step workflow.
	Step string `json:"step,omitempty"`
}

// AmqpRoutingKey is the routing key for AMQP.
//
// Follows the pattern: <service_or_domain>.<category>.<action>
type AmqpRoutingKey string

const (
	// Commands

	// NotificationCmdSendEmail is a command to send an email notification to a user.
	NotificationCmdSendEmail AmqpRoutingKey = "notification.cmd.send_email"

	// Events

	// NotificationEventEmailSent is an event that indicates that an email has been sent successfully.
	NotificationEventEmailSent AmqpRoutingKey = "notification.event.email_sent"
	// NotificationEventEmailFailed is an event that indicates that an email has failed to send.
	NotificationEventEmailFailed AmqpRoutingKey = "notification.event.email_failed"

	// In-app messaging / notifications

	// NotificationCmdFanout instructs notification-service to turn an alert/message intent into a message plus per-recipient notification rows (system/agent/event alerts, broadcasts).
	NotificationCmdFanout AmqpRoutingKey = "notification.cmd.fanout"
	// NotificationCmdSendMessage is the async chat-send path used for heavy group/broadcast fan-out.
	NotificationCmdSendMessage AmqpRoutingKey = "notification.cmd.send_message"
	// NotificationCmdAgentReply instructs notification-service to post an agent's reply into a conversation as the agent participant (attributed + linked to the producing run). Emitted by agent-service after a chat-triggered run. Carries a Phase ("start" creates the streaming row,
	// "final"/empty finalizes it) so the reply renders as one record that streams in.
	NotificationCmdAgentReply AmqpRoutingKey = "notification.cmd.agent_reply"
	// NotificationCmdAgentReplyPatch streams a partial body into an in-flight agent reply message while the run is still producing tokens. Best-effort (published straight to the exchange, not the outbox): patches are lossy by design — they carry the full accumulated body (last-write-wins) and
	// NotificationCmdAgentReply's "final" phase reconciles the persisted row.
	NotificationCmdAgentReplyPatch AmqpRoutingKey = "notification.cmd.agent_reply_patch"
	// NotificationEventDelivered carries a best-effort realtime push to the gateways (bell + live chat).
	NotificationEventDelivered AmqpRoutingKey = "notification.event.delivered"
	// NotificationEventConversationUpdated carries unread/last-message/typing updates to the gateways.
	NotificationEventConversationUpdated AmqpRoutingKey = "notification.event.conversation_updated"

	// Core

	// CoreCmdPurgeAccountData is a command to purge all account-scoped data for a deleted sandbox account.
	CoreCmdPurgeAccountData AmqpRoutingKey = "core.cmd.purge_account_data"

	// CoreCmdSeedSandbox is a command to populate a sandbox account with tutorial seed data.
	CoreCmdSeedSandbox AmqpRoutingKey = "core.cmd.seed_sandbox"

	// CoreCmdExecuteProductionStep is a command to execute production step side-effects (inventory updates, reservation management) after a batch mutation such as initialize, move, merge, or split.
	CoreCmdExecuteProductionStep AmqpRoutingKey = "core.cmd.execute_production_step"

	// CoreCmdRecalcItemBurnRate is a command to recompute an item's burn rate from its recent consumption history, off the transaction that recorded the consumption so the shared rate row is not X-locked for the length of that long transaction.
	CoreCmdRecalcItemBurnRate AmqpRoutingKey = "core.cmd.recalc_item_burn_rate"

	// CoreCmdAllocateOpenIssues is a command to allocate one bounded page of an item's open inventory issues against available receipts, resuming after a cursor the command carries and re-enqueuing a continuation while more remain, so the walk does not run inline in the scan transaction that enqueued it.
	CoreCmdAllocateOpenIssues AmqpRoutingKey = "core.cmd.allocate_open_issues"

	// CoreCmdUndoBatchScan is a command to reverse the inventory a scan recorded against a batch that has just been deleted: the receipts it produced, the issues it consumed, and the reservations it drew down. The delete itself is synchronous; this unwinds the ledger behind it.
	CoreCmdUndoBatchScan AmqpRoutingKey = "core.cmd.undo_batch_scan"

	// CoreCmdSyncStripeCustomer is a command to reconcile a customer with the account's connected Stripe integration: create the Stripe customer on first sync, or push a changed email/name/number onto the existing one. Published by customer create/update so a Stripe outage can never fail the mutation that triggered it.
	CoreCmdSyncStripeCustomer AmqpRoutingKey = "core.cmd.sync_stripe_customer"

	// Async bulk operation command routing keys ("core.cmd.<slug>") are derived from their
	// canonical identity in shared/messaging (messaging.BulkOperation.RoutingKey), the
	// single source of truth for each operation's queue, routing key, and inbox handler.

	// runs the read-only matching pass of a HubSpot backfill job out-of-band so the triggering request returns immediately.
	CoreCmdHubspotSyncPreview AmqpRoutingKey = "core.cmd.hubspot_sync_preview"

	// runs the write phase of a reviewed HubSpot backfill job.
	CoreCmdHubspotSyncExecute AmqpRoutingKey = "core.cmd.hubspot_sync_execute"

	// asks the core-service to solve and persist one production schedule version. Published by the generation cadence, which only enqueues: a solve takes minutes, and running it inside the scheduler lease would block every other account behind whichever one is solving.
	CoreCmdGenerateProductionSchedule AmqpRoutingKey = "core.cmd.generate_production_schedule"

	// indicates a schedule version was published and its first weeks frozen. Consumers notify the departments that now have a committed plan to work to.
	CoreEventProductionSchedulePublished AmqpRoutingKey = "core.event.production_schedule_published"

	// indicates a sales order was created. Consumers use it to run out-of-band side effects (e.g. syncing the order to a third-party CRM such as HubSpot) without blocking the create response.
	CoreEventSalesOrderCreated AmqpRoutingKey = "core.event.sales_order_created"

	// CoreEventSalesOrderShippingUpdated indicates a sales order's carrier, service level, or ship-to address changed. The core-service consumer re-syncs the order's existing shipment records to match, out-of-band from the update response.
	CoreEventSalesOrderShippingUpdated AmqpRoutingKey = "core.event.sales_order_shipping_updated"

	// CoreEventCustomerRegistered indicates a buyer completed registration on a seller's customer portal (a brand-new customer account or a new login joining an existing one). The notification-service consumer notifies the seller's customer-service support-route group so they can follow up.
	CoreEventCustomerRegistered AmqpRoutingKey = "core.event.customer_registered"

	// CoreEventInventoryReceived states that stock of an item became available. Allocation is one reaction to it: an issue that went short because the shelf could not cover it is filled the moment what it was waiting for arrives, rather than at whatever hour a sweep happens to run.
	CoreEventInventoryReceived AmqpRoutingKey = "core.event.inventory_received"

	// CoreEventItemCostBasisChanged states that something an item's cost is derived from moved — the unit cost of a material, or the make-up of a production step. Costing reacts by recomputing every item downstream of the change, since a cost is only as current as the inputs it was last calculated from.
	CoreEventItemCostBasisChanged AmqpRoutingKey = "core.event.item_cost_basis_changed"

	// CoreEventBatchScanned states that a batch was scanned at a station: the unit of work exists and carries the measures the operator recorded. It says nothing about what should follow, so a subscriber that wants to move inventory, credit the schedule, or notify a department binds its own queue and decides for itself.
	//
	// This supersedes CoreCmdExecuteProductionStep, which named one particular reaction and therefore needed a second message to carry seconds and waste. The scan is a single fact, so it is published once and its subscribers derive the rest.
	CoreEventBatchScanned AmqpRoutingKey = "core.event.batch_scanned"

	// Logging

	// LoggingEventRequestLogged is an event that indicates that a request has been logged.
	LoggingEventRequestLogged AmqpRoutingKey = "logging.event.request_logged"

	// Billing

	// BillingEventStripeWebhook is an event carrying a verified Stripe webhook payload for asynchronous processing.
	BillingEventStripeWebhook AmqpRoutingKey = "billing.event.stripe_webhook"

	// Agent

	// AgentCmdExecuteRun is a command to execute an agent run.
	AgentCmdExecuteRun AmqpRoutingKey = "agent.cmd.execute_run"
	// AgentCmdExecuteAction is a command to execute a proposed agent action.
	AgentCmdExecuteAction AmqpRoutingKey = "agent.cmd.execute_action"
	// AgentCmdContinueRun is a command to continue an agent run awaiting input.
	AgentCmdContinueRun AmqpRoutingKey = "agent.cmd.continue_run"
	// AgentCmdChatRun starts an agent run from a chat message: agent-service creates a chat-linked run
	// (conversation_id + trigger_message_id) and executes it. Emitted by notification-service when an agent participant's trigger policy fires.
	AgentCmdChatRun AmqpRoutingKey = "agent.cmd.chat_run"
	// AgentEventRunCompleted is an event indicating an agent run has finished.
	AgentEventRunCompleted AmqpRoutingKey = "agent.event.run_completed"
	// AgentEventRunStep is an event carrying a single run step for real-time WebSocket streaming to the frontend.
	AgentEventRunStep AmqpRoutingKey = "agent.event.run_step"

	// BillingCmdSyncSeats is a command to synchronize seat counts with the billing provider after account user changes.
	BillingCmdSyncSeats AmqpRoutingKey = "billing.cmd.sync_seats"

	// BillingCmdReportSeatChange is a command to report a seat count change to the billing provider's usage metering system.
	BillingCmdReportSeatChange AmqpRoutingKey = "billing.cmd.report_seat_change"

	// Commands the billing provider's usage metering system to record a created invoice.
	BillingCmdReportInvoiceCreated AmqpRoutingKey = "billing.cmd.report_invoice_created"

	// PlatformEventAuditLogged is an event that indicates an audit event has been produced and needs to be persisted by the platform-service.
	PlatformEventAuditLogged AmqpRoutingKey = "platform.event.audit_logged"
)
