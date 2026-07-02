package constants

// MessageSendMode is whether a create-message request is delivered immediately (or scheduled) or held as a customer-reply draft awaiting human approval.
type MessageSendMode string

const (
	// MessageSendModeSend delivers the message (immediately, or at scheduled_at).
	MessageSendModeSend MessageSendMode = "send"
	// MessageSendModeDraft creates a status-draft customer-reply proposal, held for approval rather than sent.
	MessageSendModeDraft MessageSendMode = "draft"
)

func (m MessageSendMode) IsValid() bool {
	switch m {
	case MessageSendModeSend, MessageSendModeDraft:
		return true
	default:
		return false
	}
}

func (m MessageSendMode) EnumValues() []string {
	return []string{string(MessageSendModeSend), string(MessageSendModeDraft)}
}

func (m *MessageSendMode) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
