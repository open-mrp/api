package constants

// EmailInboxStatus is whether an email inbox accepts inbound mail.
type EmailInboxStatus string

const (
	// EmailInboxStatusActive threads inbound mail into a conversation.
	EmailInboxStatusActive EmailInboxStatus = "active"
	// EmailInboxStatusDisabled keeps the inbox provisioned and its history intact, but drops inbound mail without threading it.
	EmailInboxStatusDisabled EmailInboxStatus = "disabled"
)

func (s EmailInboxStatus) IsValid() bool {
	switch s {
	case EmailInboxStatusActive, EmailInboxStatusDisabled:
		return true
	default:
		return false
	}
}

func (s EmailInboxStatus) EnumValues() []string {
	return []string{
		string(EmailInboxStatusActive),
		string(EmailInboxStatusDisabled),
	}
}
