package contracts

// AmqpMessage is the message structure for AMQP.
type AmqpMessage struct {
	UserID string `json:"userId"`
	Data   []byte `json:"data"`
}

// Routing keys - using consistent event/command patterns
const (
	// Commands
	NotificationCmdSendEmail = "notification.cmd.send_email"

	// Events
	NotificationEventEmailSent   = "notification.event.email_sent"
	NotificationEventEmailFailed = "notification.event.email_failed"

	// Logging
	LoggingEventRequestLogged = "logging.event.request_logged"
)
