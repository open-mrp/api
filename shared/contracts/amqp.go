package contracts

import "github.com/augno/api/services/auth-service/pkg/types"

// AmqpMessage is the envelope for all messages published to and consumed from
// RabbitMQ. It carries the caller's identity, the application payload, and
// tracing/idempotency metadata needed to correlate and deduplicate messages.
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

	// NotificationCmdSendEmail is a command to send an email notification to a
	// user.
	NotificationCmdSendEmail AmqpRoutingKey = "notification.cmd.send_email"

	// Events

	// NotificationEventEmailSent is an event that indicates that an email has been sent successfully.
	NotificationEventEmailSent AmqpRoutingKey = "notification.event.email_sent"
	// NotificationEventEmailFailed is an event that indicates that an email has failed to send.
	NotificationEventEmailFailed AmqpRoutingKey = "notification.event.email_failed"

	// Core

	// CoreCmdPurgeAccountData is a command to purge all account-scoped data for
	// a deleted sandbox account.
	CoreCmdPurgeAccountData AmqpRoutingKey = "core.cmd.purge_account_data"

	// CoreCmdSeedSandbox is a command to populate a sandbox account with
	// tutorial seed data.
	CoreCmdSeedSandbox AmqpRoutingKey = "core.cmd.seed_sandbox"

	// CoreCmdExecuteProductionStep is a command to execute production step
	// side-effects (inventory updates, reservation management) after a batch
	// mutation such as initialize, move, merge, or split.
	CoreCmdExecuteProductionStep AmqpRoutingKey = "core.cmd.execute_production_step"

	// CoreEventSalesOrderCreated is an event indicating a sales order was
	// created. Consumers use it to run out-of-band side effects (e.g. syncing
	// the order to a third-party CRM such as HubSpot) without blocking the
	// create response.
	CoreEventSalesOrderCreated AmqpRoutingKey = "core.event.sales_order_created"

	// Logging

	// LoggingEventRequestLogged is an event that indicates that a request has been logged.
	LoggingEventRequestLogged AmqpRoutingKey = "logging.event.request_logged"

	// Billing

	// BillingEventStripeWebhook is an event carrying a verified Stripe webhook
	// payload for asynchronous processing.
	BillingEventStripeWebhook AmqpRoutingKey = "billing.event.stripe_webhook"

	// Agent

	// AgentCmdExecuteRun is a command to execute an agent run.
	AgentCmdExecuteRun AmqpRoutingKey = "agent.cmd.execute_run"
	// AgentCmdProcessEmail is a command to process an inbound email via an agent.
	AgentCmdProcessEmail AmqpRoutingKey = "agent.cmd.process_email"
	// AgentCmdExecuteAction is a command to execute a proposed agent action.
	AgentCmdExecuteAction AmqpRoutingKey = "agent.cmd.execute_action"
	// AgentCmdContinueRun is a command to continue an agent run awaiting input.
	AgentCmdContinueRun AmqpRoutingKey = "agent.cmd.continue_run"
	// AgentEventRunCompleted is an event indicating an agent run has finished.
	AgentEventRunCompleted AmqpRoutingKey = "agent.event.run_completed"
	// AgentEventRunStep is an event carrying a single run step for real-time
	// WebSocket streaming to the frontend.
	AgentEventRunStep AmqpRoutingKey = "agent.event.run_step"

	// BillingCmdSyncSeats is a command to synchronize seat counts with the
	// billing provider after account user changes.
	BillingCmdSyncSeats AmqpRoutingKey = "billing.cmd.sync_seats"

	// BillingCmdReportSeatChange is a command to report a seat count change to
	// the billing provider's usage metering system.
	BillingCmdReportSeatChange AmqpRoutingKey = "billing.cmd.report_seat_change"

	// PlatformEventAuditLogged is an event that indicates an audit event has
	// been produced and needs to be persisted by the platform-service.
	PlatformEventAuditLogged AmqpRoutingKey = "platform.event.audit_logged"
)
