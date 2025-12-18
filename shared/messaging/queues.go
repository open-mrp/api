package messaging

const (
	NotificationCmdSendEmailQueue  = "notification_cmd_send_email"
	NotifyEmailStatusQueue         = "notify_email_status"
	NotificationEventEmailLogQueue = "notification_event_email_log"
	LoggingEventRequestLogQueue    = "logging_event_request_log"

	DeadLetterQueue = "dead_letter_queue"
)

type EmailSendData struct {
	To         []string `json:"to"`
	Subject    string   `json:"subject"`
	Body       string   `json:"body"`
	IsBodyHTML bool     `json:"is_body_html"`
	SendAs     *string  `json:"send_as,omitempty"`
	AccountID  string   `json:"account_id"`
	SentByID   *string  `json:"sent_by_id,omitempty"`
}

type EmailLogData struct {
	SesMessageID string  `json:"ses_message_id"`
	AccountID    string  `json:"account_id"`
	SentByID     *string `json:"sent_by_id,omitempty"`
	Subject      string  `json:"subject"`
	Filename     *string `json:"filename,omitempty"`
}
