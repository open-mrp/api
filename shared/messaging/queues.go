package messaging

import "github.com/augno/api/shared/constants"

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
