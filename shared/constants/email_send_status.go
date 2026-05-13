package constants

// EmailSendStatus represents the delivery lifecycle status of an email.
type EmailSendStatus string

const (
	// EmailSendStatusSent indicates the email was sent.
	EmailSendStatusSent EmailSendStatus = "sent"
	// EmailSendStatusPending indicates the email has not been sent yet.
	EmailSendStatusPending EmailSendStatus = "pending"
)

func (m EmailSendStatus) IsValid() bool {
	switch m {
	case EmailSendStatusSent, EmailSendStatusPending:
		return true
	default:
		return false
	}
}

func (m EmailSendStatus) EnumValues() []string {
	return []string{
		string(EmailSendStatusSent),
		string(EmailSendStatusPending),
	}
}
